// State Management
const state = {
    queue: [],
    isUploading: false,
    token: new URLSearchParams(window.location.search).get('token') || '',
    isAuthenticated: false,
    ws: null,
    wsReconnectDelay: 1000,
    speedWindow: {
        lastBytes: 0,
        lastTime: 0,
        speedMBps: 0
    }
};

// Initialize Application
document.addEventListener('DOMContentLoaded', () => {
    // Strip token parameter from browser address bar to protect against shoulder surfing
    if (window.location.search.includes('token=')) {
        state.isAuthenticated = true;
        window.history.replaceState({}, document.title, window.location.pathname);
    }

    initDropzone();
    initFileInput();
    initPinInput();
    checkHealth();
    verifyAuthAndInit();
});

// Verify Authentication and Initialize Services
function verifyAuthAndInit() {
    const headers = state.token ? { 'X-WarpDrop-Token': state.token } : {};
    fetch('/api/v1/files', { headers })
        .then(res => {
            if (res.status === 403 || res.status === 401) {
                showPinModal();
            } else if (res.ok) {
                state.isAuthenticated = true;
                hidePinModal();
                initWebSocket();
                return res.json().then(renderFilesList);
            }
        })
        .catch(() => {
            showPinModal();
        });
}

// 4-Digit PIN Authentication
function initPinInput() {
    const pinInput = document.getElementById('pinInput');
    if (!pinInput) return;

    pinInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            submitPinAuth();
        }
    });

    pinInput.addEventListener('input', () => {
        pinInput.classList.remove('error');
        document.getElementById('pinErrorMessage').style.display = 'none';
        if (pinInput.value.length === 4) {
            submitPinAuth();
        }
    });
}

function showPinModal() {
    const modal = document.getElementById('pinModal');
    if (modal) {
        modal.style.display = 'flex';
        const input = document.getElementById('pinInput');
        if (input) {
            input.value = '';
            input.focus();
        }
    }
}

function hidePinModal() {
    const modal = document.getElementById('pinModal');
    if (modal) modal.style.display = 'none';
}

function submitPinAuth() {
    const pinInput = document.getElementById('pinInput');
    const errorEl = document.getElementById('pinErrorMessage');
    const submitBtn = document.getElementById('pinSubmitBtn');
    const pin = pinInput.value.trim();

    if (pin.length !== 4) {
        errorEl.textContent = 'Masukkan 4-digit PIN dengan benar';
        errorEl.style.display = 'block';
        pinInput.classList.add('error');
        return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = 'MEMVERIFIKASI...';

    fetch('/api/v1/auth/pin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pin: pin })
    })
    .then(async res => {
        submitBtn.disabled = false;
        submitBtn.textContent = 'MASUK';

        if (res.status === 429) {
            startLockoutCountdown(60);
            return;
        }

        const data = await res.json();
        if (!res.ok) {
            errorEl.textContent = data.message || 'Kode PIN salah';
            errorEl.style.display = 'block';
            pinInput.classList.add('error');
            pinInput.value = '';
            pinInput.focus();
            return;
        }

        // Success
        state.isAuthenticated = true;
        if (data.token) state.token = data.token;
        hidePinModal();
        initWebSocket();
        fetchFilesList();
    })
    .catch(() => {
        submitBtn.disabled = false;
        submitBtn.textContent = 'MASUK';
        errorEl.textContent = 'Terjadi kesalahan jaringan saat verifikasi PIN';
        errorEl.style.display = 'block';
    });
}

function startLockoutCountdown(seconds) {
    const errorEl = document.getElementById('pinErrorMessage');
    const pinInput = document.getElementById('pinInput');
    const submitBtn = document.getElementById('pinSubmitBtn');

    pinInput.disabled = true;
    submitBtn.disabled = true;
    let remaining = seconds;

    const timer = setInterval(() => {
        errorEl.textContent = `Terlalu banyak percobaan gagal. IP dikunci (${remaining} detik)`;
        errorEl.style.display = 'block';
        remaining--;

        if (remaining < 0) {
            clearInterval(timer);
            pinInput.disabled = false;
            submitBtn.disabled = false;
            errorEl.style.display = 'none';
            pinInput.value = '';
            pinInput.focus();
        }
    }, 1000);
}

// WebSocket Connection & Real-Time Shared Clipboard
function initWebSocket() {
    if (!state.isAuthenticated || state.ws) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`;

    try {
        state.ws = new WebSocket(wsUrl);

        state.ws.onopen = () => {
            state.wsReconnectDelay = 1000;
        };

        state.ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'clipboard' && msg.content) {
                    addClipboardCard(msg.content, msg.timestamp || new Date().toISOString(), false);
                }
            } catch (e) {
                console.error('Gagal memproses pesan WebSocket:', e);
            }
        };

        state.ws.onclose = () => {
            state.ws = null;
            // Exponential backoff auto-reconnect
            setTimeout(() => {
                state.wsReconnectDelay = Math.min(state.wsReconnectDelay * 2, 10000);
                initWebSocket();
            }, state.wsReconnectDelay);
        };

        state.ws.onerror = () => {
            if (state.ws) state.ws.close();
        };
    } catch (err) {
        console.error('Gagal menghubungkan WebSocket:', err);
    }
}

function sendClipboardText() {
    const input = document.getElementById('clipboardInput');
    const content = input.value.trim();

    if (!content) return;

    if (content.length > 1024 * 1024) {
        alert('Panjang teks melebihi batas maksimal 1 MB');
        return;
    }

    const payload = {
        type: 'clipboard',
        content: content,
        timestamp: new Date().toISOString()
    };

    if (state.ws && state.ws.readyState === WebSocket.OPEN) {
        state.ws.send(JSON.stringify(payload));
    }

    addClipboardCard(content, payload.timestamp, true);
    input.value = '';
}

function addClipboardCard(text, timestamp, isLocal) {
    const list = document.getElementById('clipboardList');
    const emptyMsg = document.getElementById('emptyClipboardMsg');
    if (!list) return;

    if (emptyMsg) emptyMsg.style.display = 'none';

    const li = document.createElement('li');
    li.className = 'clipboard-item';

    const textDiv = document.createElement('div');
    textDiv.className = 'clipboard-item-text';
    textDiv.textContent = text;

    const metaDiv = document.createElement('div');
    metaDiv.className = 'clipboard-item-meta';

    const timeSpan = document.createElement('span');
    const date = new Date(timestamp).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    timeSpan.textContent = (isLocal ? 'Dikirim - ' : 'Diterima - ') + date;

    const copyBtn = document.createElement('button');
    copyBtn.className = 'clipboard-copy-btn';
    copyBtn.textContent = 'SALIN';
    copyBtn.onclick = () => copyTextAction(copyBtn, text);

    metaDiv.appendChild(timeSpan);
    metaDiv.appendChild(copyBtn);

    li.appendChild(textDiv);
    li.appendChild(metaDiv);

    list.insertBefore(li, list.firstChild);
}

function copyTextAction(btn, text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => markCopied(btn)).catch(() => fallbackCopy(btn, text));
    } else {
        fallbackCopy(btn, text);
    }
}

function fallbackCopy(btn, text) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    document.body.appendChild(textarea);
    textarea.select();
    try {
        document.execCommand('copy');
        markCopied(btn);
    } catch (e) {
        console.error('Gagal menyalin teks:', e);
    }
    document.body.removeChild(textarea);
}

function markCopied(btn) {
    const originalText = btn.textContent;
    btn.textContent = 'TERSALIN';
    btn.classList.add('copied');
    setTimeout(() => {
        btn.textContent = originalText;
        btn.classList.remove('copied');
    }, 2000);
}

function clearClipboardHistory() {
    const list = document.getElementById('clipboardList');
    const emptyMsg = document.getElementById('emptyClipboardMsg');
    if (list) list.innerHTML = '';
    if (emptyMsg) emptyMsg.style.display = 'block';
}

// "Unduh Semua" (.ZIP) Trigger
function downloadAllZip() {
    const downloadAllUrl = '/api/v1/download-all';
    const a = document.createElement('a');
    a.href = downloadAllUrl;
    a.setAttribute('download', 'warpdrop_all.zip');
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
}

// Tab Switching
function switchTab(tabName) {
    const uploadBtn = document.getElementById('tabUploadBtn');
    const clipboardBtn = document.getElementById('tabClipboardBtn');
    const downloadBtn = document.getElementById('tabDownloadBtn');

    const uploadSec = document.getElementById('uploadSection');
    const clipboardSec = document.getElementById('clipboardSection');
    const downloadSec = document.getElementById('downloadSection');

    // Reset buttons
    uploadBtn.classList.remove('active');
    clipboardBtn.classList.remove('active');
    downloadBtn.classList.remove('active');

    // Reset sections
    uploadSec.style.display = 'none';
    clipboardSec.style.display = 'none';
    downloadSec.style.display = 'none';

    if (tabName === 'upload') {
        uploadBtn.classList.add('active');
        uploadSec.style.display = 'block';
    } else if (tabName === 'clipboard') {
        clipboardBtn.classList.add('active');
        clipboardSec.style.display = 'block';
        document.getElementById('clipboardInput').focus();
    } else if (tabName === 'download') {
        downloadBtn.classList.add('active');
        downloadSec.style.display = 'block';
        fetchFilesList();
    }
}

// Health Check
function checkHealth() {
    fetch('/api/v1/health')
        .then(res => res.json())
        .then(() => setStatusOnline(true))
        .catch(() => setStatusOnline(false));
}

function setStatusOnline(isOnline) {
    const statusWrapper = document.querySelector('.connection-status');
    const statusText = document.getElementById('statusText');
    if (isOnline) {
        statusWrapper.classList.remove('disconnected');
        statusText.textContent = 'TERHUBUNG';
    } else {
        statusWrapper.classList.add('disconnected');
        statusText.textContent = 'TERPUTUS';
    }
}

// Dropzone Initialization
function initDropzone() {
    const dropzone = document.getElementById('dropzone');

    ['dragenter', 'dragover'].forEach(eventName => {
        dropzone.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropzone.classList.add('drag-active');
        }, false);
    });

    ['dragleave', 'drop'].forEach(eventName => {
        dropzone.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropzone.classList.remove('drag-active');
        }, false);
    });

    dropzone.addEventListener('drop', (e) => {
        const dt = e.dataTransfer;
        const files = dt.files;
        handleFiles(files);
    }, false);
}

// File Input Initialization
function initFileInput() {
    const fileInput = document.getElementById('fileInput');
    fileInput.addEventListener('change', (e) => {
        handleFiles(e.target.files);
        fileInput.value = '';
    });
}

// Handle Incoming Files
function handleFiles(fileList) {
    if (!fileList || fileList.length === 0) return;

    for (let i = 0; i < fileList.length; i++) {
        const file = fileList[i];
        state.queue.push({
            id: Date.now() + '_' + i,
            file: file,
            name: file.name,
            size: file.size,
            status: 'pending'
        });
    }

    renderQueue();
    processQueue();
}

// Queue Rendering
function renderQueue() {
    const queueSection = document.getElementById('queueSection');
    const queueList = document.getElementById('queueList');
    const queueCount = document.getElementById('queueCount');

    if (state.queue.length === 0) {
        queueSection.style.display = 'none';
        return;
    }

    queueSection.style.display = 'block';
    queueCount.textContent = state.queue.length;
    queueList.innerHTML = '';

    state.queue.forEach(item => {
        const li = document.createElement('li');
        li.className = 'list-item';

        const infoDiv = document.createElement('div');
        infoDiv.className = 'item-info';

        const nameSpan = document.createElement('span');
        nameSpan.className = 'item-name';
        nameSpan.textContent = item.name;

        const sizeSpan = document.createElement('span');
        sizeSpan.className = 'item-meta';
        sizeSpan.textContent = formatBytes(item.size);

        infoDiv.appendChild(nameSpan);
        infoDiv.appendChild(sizeSpan);

        const badge = document.createElement('span');
        badge.className = 'badge badge-' + getBadgeClass(item.status);
        badge.textContent = getBadgeLabel(item.status);

        li.appendChild(infoDiv);
        li.appendChild(badge);
        queueList.appendChild(li);
    });
}

function getBadgeClass(status) {
    switch (status) {
        case 'uploading': return 'active';
        case 'success': return 'success';
        case 'error': return 'error';
        default: return 'pending';
    }
}

function getBadgeLabel(status) {
    switch (status) {
        case 'uploading': return 'MENGIRIM';
        case 'success': return 'SELESAI';
        case 'error': return 'GAGAL';
        default: return 'MENUNGGU';
    }
}

function clearCompletedQueue() {
    state.queue = state.queue.filter(item => item.status !== 'success');
    renderQueue();
}

// Queue Processing & Upload Streaming
function processQueue() {
    if (state.isUploading) return;

    const nextItem = state.queue.find(item => item.status === 'pending');
    if (!nextItem) {
        document.getElementById('speedometerCard').style.display = 'none';
        fetchFilesList();
        return;
    }

    state.isUploading = true;
    nextItem.status = 'uploading';
    renderQueue();

    // Show Speedometer Card
    const speedCard = document.getElementById('speedometerCard');
    const transferFileName = document.getElementById('transferFileName');
    const progressBar = document.getElementById('progressBar');
    const transferPercentage = document.getElementById('transferPercentage');
    const transferSpeed = document.getElementById('transferSpeed');
    const transferBytes = document.getElementById('transferBytes');
    const transferEta = document.getElementById('transferEta');

    speedCard.style.display = 'flex';
    transferFileName.textContent = nextItem.name;
    progressBar.style.width = '0%';
    transferPercentage.textContent = '0%';
    transferSpeed.textContent = '0.0 MB/s';
    transferBytes.textContent = '0 B / ' + formatBytes(nextItem.size);
    transferEta.textContent = '--';

    state.speedWindow.lastBytes = 0;
    state.speedWindow.lastTime = Date.now();
    state.speedWindow.speedMBps = 0;

    const formData = new FormData();
    formData.append('files', nextItem.file);

    const xhr = new XMLHttpRequest();
    const uploadUrl = '/api/v1/upload';
    xhr.open('POST', uploadUrl, true);
    if (state.token) {
        xhr.setRequestHeader('X-WarpDrop-Token', state.token);
    }

    xhr.upload.onprogress = (e) => {
        if (!e.lengthComputable) return;

        const percent = Math.round((e.loaded / e.total) * 100);
        progressBar.style.width = percent + '%';
        transferPercentage.textContent = percent + '%';
        transferBytes.textContent = formatBytes(e.loaded) + ' / ' + formatBytes(e.total);

        // Calculate moving speed
        const now = Date.now();
        const deltaTime = (now - state.speedWindow.lastTime) / 1000;
        if (deltaTime >= 0.25) {
            const deltaBytes = e.loaded - state.speedWindow.lastBytes;
            const currentSpeedMBps = (deltaBytes / (1024 * 1024)) / deltaTime;
            state.speedWindow.speedMBps = currentSpeedMBps;
            state.speedWindow.lastBytes = e.loaded;
            state.speedWindow.lastTime = now;

            transferSpeed.textContent = currentSpeedMBps.toFixed(1) + ' MB/s';

            // Calculate ETA
            if (currentSpeedMBps > 0) {
                const remainingBytes = e.total - e.loaded;
                const remainingSecs = Math.round((remainingBytes / (1024 * 1024)) / currentSpeedMBps);
                transferEta.textContent = remainingSecs + ' dtk';
            }
        }
    };

    xhr.onload = () => {
        state.isUploading = false;
        if (xhr.status >= 200 && xhr.status < 300) {
            nextItem.status = 'success';
        } else {
            nextItem.status = 'error';
        }
        renderQueue();
        processQueue();
    };

    xhr.onerror = () => {
        state.isUploading = false;
        nextItem.status = 'error';
        renderQueue();
        processQueue();
    };

    xhr.send(formData);
}

// Fetch Files List From Server
function fetchFilesList() {
    const url = '/api/v1/files';
    const headers = state.token ? { 'X-WarpDrop-Token': state.token } : {};

    fetch(url, { headers })
        .then(res => {
            if (res.status === 403 || res.status === 401) {
                showPinModal();
                throw new Error('Autentikasi diperlukan');
            }
            if (!res.ok) throw new Error('Gagal mengambil daftar berkas');
            return res.json();
        })
        .then(renderFilesList)
        .catch(err => {
            console.error('Error fetching files:', err);
        });
}

function renderFilesList(files) {
    const filesList = document.getElementById('filesList');
    const emptyMsg = document.getElementById('emptyFilesMessage');
    if (!filesList) return;

    filesList.innerHTML = '';
    if (!files || files.length === 0) {
        if (emptyMsg) emptyMsg.style.display = 'block';
        return;
    }

    if (emptyMsg) emptyMsg.style.display = 'none';
    files.forEach(file => {
        const li = document.createElement('li');
        li.className = 'list-item';

        const infoDiv = document.createElement('div');
        infoDiv.className = 'item-info';

        const nameSpan = document.createElement('span');
        nameSpan.className = 'item-name';
        nameSpan.textContent = file.name;

        const metaSpan = document.createElement('span');
        metaSpan.className = 'item-meta';
        const date = new Date(file.mod_time).toLocaleString('id-ID');
        metaSpan.textContent = file.human_size + ' - ' + date;

        infoDiv.appendChild(nameSpan);
        infoDiv.appendChild(metaSpan);

        const downloadLink = document.createElement('a');
        downloadLink.className = 'btn btn-secondary btn-sm';
        downloadLink.textContent = 'UNDUH';
        const downloadUrl = '/api/v1/download?file=' + encodeURIComponent(file.name);
        downloadLink.href = downloadUrl;
        downloadLink.setAttribute('download', file.name);

        li.appendChild(infoDiv);
        li.appendChild(downloadLink);
        filesList.appendChild(li);
    });
}

// Utility: Format bytes into human-readable string
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}
