async function checkAuthStatus() {
    showLoading();
    try {
        const response = await fetch('/api/user');
        const userData = await response.json();
        
        hideLoading();
        
        if (userData.authenticated) {
            showAuthenticatedSection(userData);
        } else {
            showAuthSection();
        }
    } catch (error) {
        hideLoading();
        showError('Failed to check authentication status');
        showAuthSection();
    }
}

function showAuthSection() {
    document.getElementById('auth-section').style.display = 'block';
    document.getElementById('authenticated-section').style.display = 'none';
    
    document.getElementById('spotify-login').onclick = function() {
        window.location.href = '/auth/spotify';
    };
}

function showAuthenticatedSection(userData) {
    document.getElementById('auth-section').style.display = 'none';
    document.getElementById('authenticated-section').style.display = 'block';
    
    // Populate playlists dropdown
    const playlistSelect = document.getElementById('playlist-select');
    playlistSelect.innerHTML = '<option value="">Select a playlist...</option>';
    
    if (userData.playlists && userData.playlists.length > 0) {
        userData.playlists.forEach(playlist => {
            const option = document.createElement('option');
            option.value = playlist.id;
            option.textContent = playlist.name;
            playlistSelect.appendChild(option);
        });
    } else {
        const option = document.createElement('option');
        option.textContent = 'No playlists found';
        option.disabled = true;
        playlistSelect.appendChild(option);
    }
}

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

function hideError() {
    document.getElementById('error-message').style.display = 'none';
}