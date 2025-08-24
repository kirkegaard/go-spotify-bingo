document.addEventListener('DOMContentLoaded', function() {
    const createGameForm = document.getElementById('create-game-form');
    const joinGameForm = document.getElementById('join-game-form');
    
    if (createGameForm) {
        createGameForm.addEventListener('submit', handleCreateGame);
    }
    
    if (joinGameForm) {
        joinGameForm.addEventListener('submit', handleJoinGame);
    }
});

async function handleCreateGame(event) {
    event.preventDefault();
    hideError();
    showLoading();
    
    const playlistSelect = document.getElementById('playlist-select');
    const playlistUrl = document.getElementById('playlist-url').value.trim();
    const playerCount = parseInt(document.getElementById('player-count').value);
    const platesPerPlayer = parseInt(document.getElementById('plates-per-player').value);
    const contentType = document.getElementById('content-type').value;
    
    const requestData = {
        player_count: playerCount,
        plates_per_player: platesPerPlayer,
        content_type: contentType
    };
    
    if (playlistUrl) {
        requestData.playlist_url = playlistUrl;
    } else if (playlistSelect.value) {
        requestData.playlist_id = playlistSelect.value;
    } else {
        hideLoading();
        showError('Please select a playlist or enter a playlist URL');
        return;
    }
    
    try {
        const response = await fetch('/api/games', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestData)
        });
        
        hideLoading();
        
        if (response.ok) {
            const gameData = await response.json();
            // Redirect to game view
            window.location.href = `/game-view.html?code=${gameData.game_code}`;
        } else {
            const errorText = await response.text();
            showError(errorText || 'Failed to create game');
        }
    } catch (error) {
        hideLoading();
        showError('Network error: ' + error.message);
    }
}

async function handleJoinGame(event) {
    event.preventDefault();
    hideError();
    showLoading();
    
    const gameCode = document.getElementById('game-code').value.trim();
    
    if (!gameCode || gameCode.length !== 6) {
        hideLoading();
        showError('Please enter a valid 6-digit game code');
        return;
    }
    
    try {
        const response = await fetch(`/api/games/join?code=${gameCode}`);
        
        hideLoading();
        
        if (response.ok) {
            const gameData = await response.json();
            // Redirect to game view
            window.location.href = `/game-view.html?code=${gameData.game_code}`;
        } else {
            const errorText = await response.text();
            showError(errorText || 'Failed to join game');
        }
    } catch (error) {
        hideLoading();
        showError('Network error: ' + error.message);
    }
}

// Utility functions (if not already defined in auth.js)
if (typeof showLoading === 'undefined') {
    function showLoading() {
        document.getElementById('loading').style.display = 'block';
    }
}

if (typeof hideLoading === 'undefined') {
    function hideLoading() {
        document.getElementById('loading').style.display = 'none';
    }
}

if (typeof showError === 'undefined') {
    function showError(message) {
        const errorDiv = document.getElementById('error-message');
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
        
        setTimeout(() => {
            errorDiv.style.display = 'none';
        }, 5000);
    }
}

if (typeof hideError === 'undefined') {
    function hideError() {
        document.getElementById('error-message').style.display = 'none';
    }
}