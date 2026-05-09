const API_URL = 'http://localhost:8080/api/v1';

// Map management
class App {
    constructor() {
        this.currentPosition = { latitude: 50.4501, longitude: 30.5234 };
        this.selectedLocation = null;
        this.currentStep = 1;
        this.isProcessing = false;

        // Auth state
        this.captureTokenFromUrl();
        this.token = this.getToken();
        this.user = this.getUser();

        // Map will be initialized when showing the window
        this.initEventListeners();
        this.checkAuth();
    }

    initMap() {
        if (this.map) return; // Already initialized

        // L is global from the script tag in index.html
        this.map = L.map('map').setView([this.currentPosition.latitude, this.currentPosition.longitude], 13);

        // Using OpenStreetMap standard tiles (free, no API key required)
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 19,
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors | Powered by <a href="https://www.geoapify.com/">Geoapify</a>'
        }).addTo(this.map);

        this.routeLayer = L.layerGroup().addTo(this.map);

        console.log('Map initialized successfully');

        // Fix for map resizing issue
        this.resizeObserver = new ResizeObserver(() => {
            this.map.invalidateSize();
        });
        this.resizeObserver.observe(document.getElementById('map'));

        // Ensure initial size is correct
        setTimeout(() => this.map.invalidateSize(), 200);
    }

    initEventListeners() {
        // Step Transitions
        // Auto-resize for textarea
        const prefInput = document.getElementById('prefInput');
        if (prefInput) {
            prefInput.addEventListener('input', function () {
                this.style.height = 'auto';
                this.style.height = (this.scrollHeight) + 'px';
            });
        }

        document.getElementById('nextToRadiusBtn')?.addEventListener('click', () => {
            const pref = document.getElementById('prefInput').value;
            if (pref.length < 3) {
                this.showToast("Будь ласка, введіть хоча б 3 символи", 'error');
                return;
            }
            this.showStep(2);
        });

        document.getElementById('backToPrefBtn')?.addEventListener('click', () => this.showStep(1));

        document.getElementById('nextToMapBtn')?.addEventListener('click', () => {
            this.showStep(3);
            this.enterSelectionMode();
        });

        document.getElementById('backToRadiusBtn')?.addEventListener('click', () => {
            this.exitSelectionMode();
            this.showStep(2);
        });

        // Slider update
        const slider = document.getElementById('radiusSlider');
        const radiusVal = document.getElementById('radiusVal');
        slider?.addEventListener('input', (e) => {
            if (radiusVal) radiusVal.innerText = e.target.value + "м";
        });

        // Form submissions
        document.getElementById('sendBtn')?.addEventListener('click', () => this.handleRouteRequest());

        // Navigation to Add Place
        document.getElementById('addPlaceBtn')?.addEventListener('click', () => {
            const center = this.map.getCenter();
            window.location.href = `add-place.html?lat=${center.lat.toFixed(6)}&lng=${center.lng.toFixed(6)}`;
        });

        // Back Button
        document.getElementById('backBtn')?.addEventListener('click', () => {
            this.resetSteps();
            const appWin = document.querySelector('.app-window');
            appWin.classList.remove('active', 'app-ready', 'selection-mode');
            setTimeout(() => {
                appWin.style.display = 'none';
                document.getElementById('landing-container').style.opacity = '1';
                document.getElementById('landing-container').style.pointerEvents = 'all';
            }, 500);
        });

        // Auth Menu
        document.getElementById('loginMenuBtn')?.addEventListener('click', (e) => {
            e.preventDefault();
            window.location.href = `${API_URL}/auth/login`;
        });

        document.getElementById('logoutMenuBtn')?.addEventListener('click', (e) => {
            e.preventDefault();
            this.logout();
        });

        // Burger Menu Toggle
        const menuIcon = document.getElementById('mainMenuIcon');
        const dropdown = document.getElementById('mainMenuDropdown');
        if (menuIcon && dropdown) {
            menuIcon.addEventListener('click', (e) => {
                e.stopPropagation();
                dropdown.classList.toggle('active');
            });

            // Close on click outside
            document.addEventListener('click', () => {
                dropdown.classList.remove('active');
            });

            // Close when any menu item is clicked
            dropdown.querySelectorAll('.menu-item').forEach(item => {
                item.addEventListener('click', () => {
                    dropdown.classList.remove('active');
                });
            });
        }
    }

    // Auth Methods
    getToken() { return localStorage.getItem('auth_token'); }
    
    getUser() {
        const user = localStorage.getItem('auth_user');
        return user ? JSON.parse(user) : null;
    }

    captureTokenFromUrl() {
        const urlParams = new URLSearchParams(window.location.search);
        const token = urlParams.get('token');
        if (token) {
            localStorage.setItem('auth_token', token);
            // Clean URL
            window.history.replaceState({}, document.title, window.location.pathname);
        }
    }

    async checkAuth() {
        this.token = this.getToken();
        
        if (this.token) {
            try {
                const res = await fetch(`${API_URL}/auth/user-info`, {
                    headers: { 'Authorization': `Bearer ${this.token}` }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.user = { username: data.claims.preferred_username || data.claims.name || 'User' };
                    localStorage.setItem('auth_user', JSON.stringify(this.user));
                } else {
                    this.logout(false);
                }
            } catch (e) {
                console.error("Failed to fetch user info", e);
            }
        } else {
            this.user = null;
        }

        const loginBtn = document.getElementById('loginMenuBtn');
        const logoutBtn = document.getElementById('logoutMenuBtn');
        const usernameSpan = document.getElementById('menuUsername');

        if (this.token && this.user) {
            if (loginBtn) loginBtn.style.display = 'none';
            if (logoutBtn) logoutBtn.style.display = 'flex';
            if (usernameSpan) usernameSpan.innerText = this.user.username;
        } else {
            if (loginBtn) loginBtn.style.display = 'flex';
            if (logoutBtn) logoutBtn.style.display = 'none';
        }
    }

    setAuth(token, user) {
        localStorage.setItem('auth_token', token);
        localStorage.setItem('auth_user', JSON.stringify(user));
        this.checkAuth();
    }

    logout(showToast = true) {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
        this.token = null;
        this.user = null;
        this.checkAuth();
        if (showToast) this.showToast('Ви вийшли з аккаунта', 'info');
    }

    showStep(step) {
        document.querySelectorAll('.form-step').forEach(s => s.classList.remove('active'));
        if (step === 1) {
            document.getElementById('step-preferences')?.classList.add('active');
        } else if (step === 2) {
            document.getElementById('step-radius')?.classList.add('active');
        }
        this.currentStep = step;
    }

    resetSteps() {
        this.showStep(1);
        this.selectedLocation = null;
        const coordDisplay = document.getElementById('selected-coords');
        if (coordDisplay) coordDisplay.innerText = "Координати не обрано";
        document.getElementById('sendBtn').disabled = true;
        if (this.startMarker) {
            this.map.removeLayer(this.startMarker);
            this.startMarker = null;
        }
        if (this.routeLayer) {
            this.routeLayer.clearLayers();
        }
    }

    enterSelectionMode() {
        // Hide landing UI
        const landing = document.getElementById('landing-container');
        landing.style.opacity = '0';
        landing.style.pointerEvents = 'none';

        const appWindow = document.querySelector('.app-window');
        appWindow.style.display = 'flex';
        appWindow.classList.add('active');
        appWindow.classList.add('selection-mode');

        if (!this.map) {
            this.initMap();
        }

        this.map.on('click', (e) => this.handleMapClick(e));
        setTimeout(() => this.map.invalidateSize(), 200);
    }

    exitSelectionMode() {
        // Restore landing UI
        const landing = document.getElementById('landing-container');
        landing.style.opacity = '1';
        landing.style.pointerEvents = 'all';

        const appWindow = document.querySelector('.app-window');
        appWindow.classList.remove('active');
        appWindow.classList.remove('selection-mode');
        this.map.off('click');
        setTimeout(() => {
            appWindow.style.display = 'none';
        }, 500);
    }

    handleMapClick(e) {
        if (this.isProcessing) return;

        const { lat, lng } = e.latlng;
        this.selectedLocation = { latitude: lat, longitude: lng };

        if (this.startMarker) {
            this.startMarker.setLatLng(e.latlng);
        } else {
            this.startMarker = L.marker(e.latlng, {
                icon: this.createCombinedIcon("Старт")
            }).addTo(this.map);
        }

        const coordDisplay = document.getElementById('selected-coords');
        if (coordDisplay) coordDisplay.innerText = `${lat.toFixed(5)}, ${lng.toFixed(5)}`;

        const sendBtn = document.getElementById('sendBtn');
        if (sendBtn) sendBtn.disabled = false;
    }

    createCombinedIcon(label) {
        return L.divIcon({
            className: 'custom-div-icon',
            html: `<div class="marker-text">${label}</div><img src="./assets/icon_image.png" style="width:40px; height:40px;">`,
            iconSize: [40, 40],
            iconAnchor: [20, 40]
        });
    }

    async handleRouteRequest() {
        const btn = document.getElementById('sendBtn');
        const pref = document.getElementById('prefInput').value;
        const radius = parseInt(document.getElementById('radiusSlider').value);

        if (!this.selectedLocation) {
            this.showToast("Будь ласка, оберіть точку на карті.", 'error');
            return;
        }

        this.isProcessing = true;
        btn.disabled = true;
        btn.innerText = "Аналіз запиту...";

        this.currentUserLocation = this.selectedLocation;
        this.currentRadius = radius;

        // Final transition to full app view
        document.querySelector('.app-window').classList.remove('selection-mode');
        document.getElementById('landing-container').style.opacity = '0';
        document.getElementById('landing-container').style.pointerEvents = 'none';

        await this.analyzeRequest(pref);
    }

    async analyzeRequest(pref) {
        try {
            const response = await fetch('http://localhost:8080/api/v1/routes/analyze', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ preferences: pref })
            });

            const tagsSegments = await response.json();

            if (!response.ok) {
                this.showToast(this.translateError(tagsSegments.error), 'error');
                this.resetRouteBtn();
                return;
            }

            if (!tagsSegments || tagsSegments.length === 0) {
                this.showToast("Не вдалося визначити теги.", 'error');
                this.resetRouteBtn();
                return;
            }

            // Show Tag Editor with Mask
            const mask = document.getElementById('tag-editor-mask');
            const appWin = document.querySelector('.app-window');
            mask.classList.add('active');
            mask.classList.remove('hidden');
            appWin.classList.add('tags-active');

            this.renderTagEditor(tagsSegments);

        } catch (err) {
            console.error(err);
            this.showToast("Помилка при аналізі запиту.", 'error');
            this.resetRouteBtn();
        }
    }

    renderTagEditor(segments) {
        const container = document.getElementById('tags-container');
        container.innerHTML = '';

        this.currentSegments = segments; // Store for build

        let renderedIndex = 1;
        segments.forEach((tags, index) => {
            if (!tags || tags.length === 0) return;

            const group = document.createElement('div');
            group.className = 'tag-group';
            group.innerHTML = `<h4>Місце ${renderedIndex++}</h4>`;

            const list = document.createElement('div');
            list.className = 'tag-list';
            list.dataset.index = index;

            // Drag & Drop events
            list.addEventListener('dragover', e => {
                e.preventDefault();
                list.style.background = 'rgba(255,255,255,0.1)';
            });
            list.addEventListener('dragleave', () => list.style.background = 'transparent');
            list.addEventListener('drop', e => this.handleDrop(e, index));

            tags.forEach(tag => {
                const tagEl = this.createTagElement(tag);
                list.appendChild(tagEl);
            });

            group.appendChild(list);
            container.appendChild(group);
        });

        // Bind buttons
        document.getElementById('confirmRouteBtn').onclick = () => this.buildRoute();
        document.getElementById('cancelRouteBtn').onclick = () => {
            document.getElementById('tag-editor-mask').classList.remove('active');
            document.getElementById('tag-editor-mask').classList.add('hidden');
            document.querySelector('.app-window').classList.remove('tags-active');
            this.resetRouteBtn();
            this.resetSteps();
            // Return to landing page
            const appWin = document.querySelector('.app-window');
            appWin.classList.remove('active', 'app-ready', 'selection-mode');
            setTimeout(() => {
                appWin.style.display = 'none';
                document.getElementById('landing-container').style.opacity = '1';
                document.getElementById('landing-container').style.pointerEvents = 'all';
            }, 500);
        };
    }

    createTagElement(tag) {
        const el = document.createElement('div');
        el.className = 'tag-item';
        el.draggable = true;
        el.innerHTML = `<span>${tag}</span><div class="tag-remove">×</div>`;

        el.querySelector('.tag-remove').onclick = (e) => {
            e.stopPropagation();
            const list = el.parentElement;
            el.remove();

            if (list && list.children.length === 0) {
                const group = list.closest('.tag-group');
                if (group) {
                    group.remove();
                    this.updatePlaceIndices();
                }
            }
        };

        el.addEventListener('dragstart', e => {
            el.classList.add('dragging');
            e.dataTransfer.setData('text/plain', tag);
            // We need to track where it came from to remove it if moved
            this.draggedElement = el;
        });

        el.addEventListener('dragend', () => {
            el.classList.remove('dragging');
            this.draggedElement = null;
        });

        return el;
    }

    handleDrop(e, targetIndex) {
        e.preventDefault();
        e.currentTarget.style.background = 'transparent';

        if (this.draggedElement) {
            const oldList = this.draggedElement.parentElement;

            // Move the element in DOM
            e.currentTarget.appendChild(this.draggedElement);

            // Check if old list is now empty and remove group if so
            if (oldList && oldList.children.length === 0) {
                const group = oldList.closest('.tag-group');
                if (group) {
                    group.remove();
                    this.updatePlaceIndices();
                }
            }
        }
    }

    updatePlaceIndices() {
        const groups = document.querySelectorAll('.tag-group');
        groups.forEach((group, index) => {
            const title = group.querySelector('h4');
            if (title) {
                title.innerText = `Місце ${index + 1}`;
            }
        });
    }

    async buildRoute() {
        const container = document.getElementById('tags-container');
        const confirmBtn = document.getElementById('confirmRouteBtn');

        // Reconstruct segments from DOM
        const lists = container.querySelectorAll('.tag-list');
        const newSegments = [];

        lists.forEach(list => {
            const tags = [];
            list.querySelectorAll('.tag-item span').forEach(span => tags.push(span.innerText));
            newSegments.push(tags);
        });

        confirmBtn.disabled = true;
        confirmBtn.innerText = "Будуємо...";

        try {
            const payload = {
                tags: newSegments,
                location: this.currentUserLocation,
                radius: this.currentRadius
            };

            await this.sendToRouteBackend(payload);

            // Clean up UI
            document.getElementById('tag-editor-mask').classList.remove('active');
            document.getElementById('tag-editor-mask').classList.add('hidden');
            document.querySelector('.app-window').classList.remove('tags-active');
            document.querySelector('.app-window').classList.add('app-ready');

        } catch (err) {
            console.error(err);
            this.showToast("Помилка побудови маршруту", 'error');
        } finally {
            confirmBtn.disabled = false;
            confirmBtn.innerText = "Побудувати маршрут";
            this.resetRouteBtn();
        }
    }

    async sendToRouteBackend(payload) {
        try {
            // Clear existing layers before fetching new data (moved to the start)
            if (this.startMarker) {
                this.map.removeLayer(this.startMarker);
                this.startMarker = null;
            }

            this.routeLayer.clearLayers();

            const response = await fetch('http://localhost:8080/api/v1/routes/build', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                const errorData = await response.json();
                this.showToast(this.translateError(errorData.error), 'error');
                return;
            }

            const data = await response.json();

            if (!data.segments || data.segments.length === 0) {
                this.showToast("Маршрут не знайдено.", 'error');
                return;
            }

            console.log("Route Data:", data);

            // Calculate total distance client-side
            let totalDistance = 0;
            data.segments.forEach(seg => {
                const feature = seg.features[0];
                if (feature && feature.properties && feature.properties.distance) {
                    totalDistance += feature.properties.distance;
                }
            });
            console.log("Calculated Total Distance:", totalDistance);

            this.totalDistanceKm = (totalDistance / 1000).toFixed(2);
            document.getElementById('route-stats').innerHTML = `Загальна відстань: ${this.totalDistanceKm} км`;
            document.getElementById('route-stats').style.display = 'block';

            // Initial view
            window.lastStatsContent = document.getElementById('route-stats').innerHTML;

            const firstSeg = data.segments[0].features[0].geometry;
            const firstCoords = firstSeg.type === "MultiLineString" ? firstSeg.coordinates[0] : firstSeg.coordinates;
            const startPt = [firstCoords[0][1], firstCoords[0][0]];

            L.marker(startPt, { icon: this.createCombinedIcon("Початок") }).addTo(this.routeLayer);

            let allPoints = [];

            data.segments.forEach((segment, index) => {
                const feature = segment.features[0];
                const placeName = segment.name || "Ціль";
                const placeDesc = segment.description || "";

                // Coordinates handling
                let latLngs = [];
                if (feature.geometry.type === "MultiLineString") {
                    feature.geometry.coordinates.forEach(line => {
                        line.forEach(c => latLngs.push([c[1], c[0]]));
                    });
                } else {
                    latLngs = feature.geometry.coordinates.map(c => [c[1], c[0]]);
                }

                allPoints.push(...latLngs);

                const polyline = L.polyline(latLngs, {
                    color: index % 2 === 0 ? '#007bff' : '#a55eea',
                    weight: 6,
                    opacity: 0.8
                }).addTo(this.routeLayer);

                // Hover events for polyline (Segment Distance)
                if (feature.properties && feature.properties.distance) {
                    const segDistKm = (feature.properties.distance / 1000).toFixed(2);

                    polyline.on('mouseover', (e) => {
                        const statsDiv = document.getElementById('route-stats');
                        statsDiv.innerHTML = `Відстань між точками: ${segDistKm} км`;
                        e.target.setStyle({ weight: 9, opacity: 1 });
                    });

                    polyline.on('mouseout', (e) => {
                        const statsDiv = document.getElementById('route-stats');
                        statsDiv.innerHTML = `Загальна відстань: ${this.totalDistanceKm} км`;
                        e.target.setStyle({ weight: 6, opacity: 0.8 });
                    });
                }

                // Place Marker
                const marker = L.marker(latLngs[latLngs.length - 1], { icon: this.createCombinedIcon(placeName) });
                marker.addTo(this.routeLayer);

                // Hover events for marker (place info)
                marker.on('mouseover', () => {
                    this.showPlaceInfo(placeName, placeDesc);
                });

                marker.on('mouseout', () => {
                    this.restorePlaceInfo();
                });
            });

            if (allPoints.length > 0) {
                this.map.fitBounds(L.latLngBounds(allPoints), { padding: [50, 50] });
            }

        } catch (err) {
            console.error(err);
            throw err;
        }
    }

    showPlaceInfo(name, desc) {
        const infoWin = document.getElementById('info-window');
        if (!this.originalInfoContent) {
            this.originalInfoContent = infoWin.innerHTML;
        }

        const descHtml = `
            <h2 style="margin-top: 0;">${name}</h2>
            <p style="color: #ccc; font-size: 0.95rem; line-height: 1.5;">${desc || "Опис відсутній"}</p>
            <div id="route-stats" class="stats-box" style="display:block">Загальна відстань: ${this.totalDistanceKm} км</div>
            <button id="backBtn" class="secondary-btn" style="width: 100%;">Новий пошук</button>
        `;
        infoWin.innerHTML = descHtml;
        this.bindBackButton();
    }

    restorePlaceInfo() {
        if (this.originalInfoContent) {
            document.getElementById('info-window').innerHTML = this.originalInfoContent;
            this.bindBackButton();
        }
    }

    bindBackButton() {
        const backBtn = document.getElementById('backBtn');
        if (backBtn) {
            backBtn.onclick = () => {
                this.resetSteps();
                const appWin = document.querySelector('.app-window');
                appWin.classList.remove('active', 'app-ready', 'selection-mode');
                setTimeout(() => {
                    appWin.style.display = 'none';
                    document.getElementById('landing-container').style.opacity = '1';
                    document.getElementById('landing-container').style.pointerEvents = 'all';
                }, 500);
            };
        }
    }

    resetRouteBtn() {
        this.isProcessing = false;
        const btn = document.getElementById('sendBtn');
        if (btn) {
            btn.disabled = false;
            btn.innerText = "Знайти маршрут";
        }
    }

    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const icon = type === 'error' ? '!' : 'i';

        toast.innerHTML = `
            <div class="toast-icon">${icon}</div>
            <div class="toast-message">${message}</div>
        `;

        container.appendChild(toast);

        setTimeout(() => {
            toast.style.animation = 'toastOut 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards';
            setTimeout(() => {
                toast.remove();
            }, 400);
        }, 4000);
    }

    translateError(errorMsg) {
        if (!errorMsg) return "Виникла невідома помилка";

        if (errorMsg.includes("failed to identify your preferences")) {
            return "Не вдалося визначити вподобання у вашому запиті. Спробуйте інші слова (наприклад, 'кава', 'парк')";
        }
        if (errorMsg.includes("no places found for the selected categories")) {
            return "Не знайдено місць за вашим запитом у вказаному радіусі. Спробуйте збільшити радіус або змінити запит.";
        }

        return errorMsg;
    }
}

// Initialize the app
new App();
