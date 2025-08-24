let currentGameData = null;
let allPlatesData = null;
let isViewingAllPlates = false;

async function loadGame(gameCode) {
    showLoading();
    
    try {
        const response = await fetch(`/api/games/join?code=${gameCode}`);
        
        if (response.ok) {
            currentGameData = await response.json();
            displayGame(currentGameData);
        } else {
            const errorText = await response.text();
            showError(errorText || 'Failed to load game');
        }
    } catch (error) {
        showError('Network error: ' + error.message);
    } finally {
        hideLoading();
    }
}

function displayGame(gameData) {
    // Update game info
    document.getElementById('game-code').textContent = gameData.game_code;
    document.getElementById('playlist-name').textContent = gameData.playlist_name || 'Unknown Playlist';
    
    // Generate plates
    const platesContainer = document.querySelector('.plates-grid');
    platesContainer.innerHTML = '';
    
    gameData.plates.forEach((plate, index) => {
        const plateElement = createPlateElement(plate, index + 1);
        platesContainer.appendChild(plateElement);
    });
    
    // Check if user is creator and show creator-only buttons
    checkIfCreator(gameData.game_code);
    
    // Set up event listeners
    setupEventListeners(gameData.game_code);
}

function createPlateElement(plate, plateNumber, playerName = null) {
    const plateDiv = document.createElement('div');
    plateDiv.className = 'bingo-plate';
    plateDiv.innerHTML = `
        <div class="plate-header">
            <div class="plate-title">BINGO</div>
            <div class="plate-number">${playerName ? `${playerName} - ` : ''}Plate ${plateNumber}</div>
        </div>
        <div class="bingo-grid" id="grid-${plate.plate_number}">
        </div>
    `;
    
    const gridElement = plateDiv.querySelector('.bingo-grid');
    
    // Generate the 3x9 grid
    for (let row = 0; row < 3; row++) {
        for (let col = 0; col < 9; col++) {
            const cellDiv = document.createElement('div');
            cellDiv.className = 'bingo-cell';
            
            const field = plate.fields.grid[row][col];
            
            if (field && field.content) {
                cellDiv.textContent = field.content;
                cellDiv.dataset.type = field.type;
                cellDiv.dataset.row = row;
                cellDiv.dataset.col = col;
                
                if (field.marked) {
                    cellDiv.classList.add('marked');
                }
                
                // Add click event for marking/unmarking
                cellDiv.addEventListener('click', function() {
                    this.classList.toggle('marked');
                });
            } else {
                cellDiv.classList.add('empty');
            }
            
            gridElement.appendChild(cellDiv);
        }
    }
    
    return plateDiv;
}

function setupEventListeners(gameCode) {
    // Print plates button
    document.getElementById('print-plates').addEventListener('click', function() {
        window.print();
    });
    
    // Print all plates button (creator only)
    document.getElementById('print-all-plates').addEventListener('click', function() {
        if (allPlatesData) {
            displayAllPlates();
            setTimeout(() => window.print(), 500);
        }
    });
    
    // View all plates button (creator only)
    document.getElementById('view-all-plates').addEventListener('click', function() {
        if (isViewingAllPlates) {
            displayGame(currentGameData);
            this.innerHTML = `
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 6px;">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                    <circle cx="9" cy="7" r="4"/>
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                    <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                </svg>
                View All Plates
            `;
            document.getElementById('print-plates').innerHTML = `
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 6px;">
                    <polyline points="6,9 6,2 18,2 18,9"/>
                    <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/>
                    <rect x="6" y="14" width="12" height="8"/>
                </svg>
                Print My Plates
            `;
            isViewingAllPlates = false;
        } else {
            loadAllPlates(gameCode);
        }
    });
    
    // Share game button
    document.getElementById('share-game').addEventListener('click', function() {
        showShareModal(gameCode);
    });
    
    // Modal close button
    document.querySelector('.close').addEventListener('click', function() {
        document.getElementById('share-modal').style.display = 'none';
    });
    
    // Close modal when clicking outside
    document.getElementById('share-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            this.style.display = 'none';
        }
    });
    
    // Copy buttons
    document.getElementById('copy-code').addEventListener('click', function() {
        copyToClipboard(document.getElementById('share-code').value, 'Game code copied!');
    });
    
    document.getElementById('copy-url').addEventListener('click', function() {
        copyToClipboard(document.getElementById('share-url').value, 'URL copied!');
    });
}

function showShareModal(gameCode) {
    const modal = document.getElementById('share-modal');
    const shareUrl = `${window.location.origin}/game-view.html?code=${gameCode}`;
    
    document.getElementById('share-code').value = gameCode;
    document.getElementById('share-url').value = shareUrl;
    
    // Generate QR code (simple text for now, could use a QR code library)
    const qrDiv = document.getElementById('qr-code');
    qrDiv.innerHTML = `
        <p><strong>QR Code:</strong></p>
        <div style="background: #f0f0f0; padding: 20px; text-align: center; border-radius: 8px; margin-top: 10px;">
            <p style="font-family: monospace; font-size: 12px; word-break: break-all;">${shareUrl}</p>
            <p><small>Use a QR code generator with this URL</small></p>
        </div>
    `;
    
    modal.style.display = 'block';
}

async function copyToClipboard(text, successMessage) {
    try {
        await navigator.clipboard.writeText(text);
        showSuccess(successMessage);
    } catch (err) {
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        try {
            document.execCommand('copy');
            showSuccess(successMessage);
        } catch (fallbackErr) {
            showError('Failed to copy to clipboard');
        }
        document.body.removeChild(textArea);
    }
}

function showSuccess(message) {
    // Create a temporary success message
    const successDiv = document.createElement('div');
    successDiv.className = 'success';
    successDiv.textContent = message;
    successDiv.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #1DB954;
        color: white;
        padding: 12px 20px;
        border-radius: 8px;
        z-index: 1001;
        animation: slideIn 0.3s ease;
    `;
    
    document.body.appendChild(successDiv);
    
    setTimeout(() => {
        successDiv.remove();
    }, 3000);
}

// Utility functions
function showLoading() {
    document.getElementById('loading').style.display = 'block';
}

function hideLoading() {
    document.getElementById('loading').style.display = 'none';
}

function showError(message) {
    const errorDiv = document.getElementById('error-message');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
    
    setTimeout(() => {
        errorDiv.style.display = 'none';
    }, 5000);
}

async function checkIfCreator(gameCode) {
    try {
        const response = await fetch(`/api/games/all-plates?code=${gameCode}`);
        if (response.ok) {
            // User is creator, show creator buttons
            document.getElementById('print-all-plates').style.display = 'inline-block';
            document.getElementById('view-all-plates').style.display = 'inline-block';
        }
    } catch (error) {
        // User is not creator or error occurred, hide buttons
        document.getElementById('print-all-plates').style.display = 'none';
        document.getElementById('view-all-plates').style.display = 'none';
    }
}

async function loadAllPlates(gameCode) {
    showLoading();
    
    try {
        const response = await fetch(`/api/games/all-plates?code=${gameCode}`);
        
        if (response.ok) {
            allPlatesData = await response.json();
            displayAllPlates();
            document.getElementById('view-all-plates').innerHTML = `
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 6px;">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                    <circle cx="12" cy="7" r="4"/>
                </svg>
                View My Plates
            `;
            document.getElementById('print-plates').innerHTML = `
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 6px;">
                    <polyline points="6,9 6,2 18,2 18,9"/>
                    <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/>
                    <rect x="6" y="14" width="12" height="8"/>
                </svg>
                Print All Plates
            `;
            isViewingAllPlates = true;
        } else {
            showError('Failed to load all plates');
        }
    } catch (error) {
        showError('Network error: ' + error.message);
    } finally {
        hideLoading();
    }
}

function displayAllPlates() {
    if (!allPlatesData) return;
    
    // Update game info
    document.getElementById('game-code').textContent = allPlatesData.game_code;
    document.getElementById('playlist-name').textContent = allPlatesData.playlist_name || 'Unknown Playlist';
    
    // Generate all plates
    const platesContainer = document.querySelector('.plates-grid');
    platesContainer.innerHTML = '';
    
    allPlatesData.all_plates.forEach(playerPlates => {
        // Add player header
        const playerHeader = document.createElement('div');
        playerHeader.className = 'player-header';
        playerHeader.innerHTML = `<h3>${playerPlates.player_id}</h3>`;
        platesContainer.appendChild(playerHeader);
        
        // Add player's plates
        playerPlates.plates.forEach((plate, index) => {
            const plateElement = createPlateElement(plate, index + 1, playerPlates.player_id);
            platesContainer.appendChild(plateElement);
        });
    });
}