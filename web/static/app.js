// State Management
const state = {
    queue: [],
    isUploading: false,
    token: new URLSearchParams(window.location.search).get('token') || '',
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
        window.history.replaceState({}, document.title, window.location.pathname);
    }

    initDropzone();
    initFileInput();
    checkHealth();
    fetchFilesList();
});

// Tab Switching
function switchTab(tabName) {
    const uploadBtn = document.getElementById('tabUploadBtn');
    const downloadBtn = document.getElementById('tabDownloadBtn');
    const uploadSec = document.getElementById('uploadSection');
    const downloadSec = document.getElementById('downloadSection');

    if (tabName === 'upload') {
        uploadBtn.classList.add('active');
        downloadBtn.classList.remove('active');
        uploadSec.classList.add('active');
        uploadSec.style.display = 'block';
        downloadSec.style.display = 'none';
    } else {
        downloadBtn.classList.add('active');
        uploadBtn.classList.remove('active');
        downloadSec.classList.add('active');
        downloadSec.style.display = 'block';
        uploadSec.style.display = 'none';
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
        statusWrapper.classList.remove('offline');
        statusText.textContent = 'TERHUBUNG';
    } else {
        statusWrapper.classList.add('offline');
        statusText.textContent = 'TERPUTUS';
    }
}

// Dropzone and File Input Handling
function initDropzone() {
    const dropzone = document.getElementById('dropzone');

    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        dropzone.addEventListener(eventName, e => {
            e.preventDefault();
            e.stopPropagation();
        });
    });

    ['dragenter', 'dragover'].forEach(eventName => {
        dropzone.addEventListener(eventName, () => dropzone.classList.add('dragover'));
    });

    ['dragleave', 'drop'].forEach(eventName => {
        dropzone.addEventListener(eventName, () => dropzone.classList.remove('dragover'));
    });

    dropzone.addEventListener('drop', e => {
        const files = e.dataTransfer.files;
        handleFilesSelected(files);
    });
}

function initFileInput() {
    const fileInput = document.getElementById('fileInput');
    fileInput.addEventListener('change', () => {
        handleFilesSelected(fileInput.files);
        fileInput.value = '';
    });
}

function handleFilesSelected(fileList) {
    if (!fileList || fileList.length === 0) return;

    for (let i = 0; i < fileList.length; i++) {
        const file = fileList[i];
        state.queue.push({
            id: 'file_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9),
            file: file,
            name: file.name,
            size: file.size,
            status: 'pending' // pending | uploading | success | error
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

        const metaSpan = document.createElement('span');
        metaSpan.className = 'item-meta';
        metaSpan.textContent = formatBytes(item.size);

        infoDiv.appendChild(nameSpan);
        infoDiv.appendChild(metaSpan);

        const badge = document.createElement('span');
        badge.className = 'item-badge badge-' + getBadgeClass(item.status);
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
    const filesList = document.getElementById('filesList');
    const emptyMsg = document.getElementById('emptyFilesMessage');
    const url = '/api/v1/files';
    const headers = state.token ? { 'X-WarpDrop-Token': state.token } : {};

    fetch(url, { headers })
        .then(res => {
            if (!res.ok) throw new Error('Gagal mengambil daftar berkas');
            return res.json();
        })
        .then(files => {
            filesList.innerHTML = '';
            if (!files || files.length === 0) {
                emptyMsg.style.display = 'block';
                return;
            }

            emptyMsg.style.display = 'none';
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
        })
        .catch(err => {
            console.error('Error fetching files:', err);
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
