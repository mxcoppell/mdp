// mdp frontend
(function() {
    'use strict';

    var config = window.__CONFIG__;
    var ws = null;
    var reconnectDelay = 500;
    var MAX_RECONNECT_DELAY = 5000;

    // Search state
    var searchMatches = [];
    var searchIndex = -1;

    // Heading navigation state
    var headings = [];

    // Vim g-key state
    var lastKeyWasG = false;
    var gKeyTimer = null;

    // Zoom state
    var zoomLevels = [75, 90, 100, 110, 125, 150];
    var zoomIndex = 2; // 100%

    // DOM elements
    var contentArea = document.getElementById('content-area');
    var content = document.getElementById('content');
    var tocSidebar = document.getElementById('toc-sidebar');
    var tocList = document.getElementById('toc-list');
    var toolbar = document.getElementById('toolbar');
    var searchBar = document.getElementById('search-bar');
    var searchInput = document.getElementById('search-input');
    var searchCount = document.getElementById('search-count');
    var disconnectedBanner = document.getElementById('disconnected-banner');
    var helpOverlay = document.getElementById('help-overlay');
    var statusInfo = document.getElementById('status-info');
    var statusScroll = document.getElementById('status-scroll');
    var dragRegion = document.getElementById('drag-region');

    // ─── Theme ────────────────────────────────────────────────
    var themeOrder = ['system', 'light', 'dark'];
    var currentThemeIndex = themeOrder.indexOf(
        localStorage.getItem('mdp-theme') || config.theme
    );
    if (currentThemeIndex === -1) currentThemeIndex = 0;

    function applyTheme() {
        var theme = themeOrder[currentThemeIndex];
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('mdp-theme', theme);
    }

    function getEffectiveThemeForIndex(idx) {
        var theme = themeOrder[idx];
        if (theme === 'system') {
            return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        }
        return theme;
    }

    function cycleTheme() {
        var effectiveBefore = getEffectiveThemeForIndex(currentThemeIndex);
        // Skip themes that look the same as the current one
        for (var attempt = 0; attempt < themeOrder.length; attempt++) {
            currentThemeIndex = (currentThemeIndex + 1) % themeOrder.length;
            if (getEffectiveThemeForIndex(currentThemeIndex) !== effectiveBefore) break;
        }
        applyTheme();
    }

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
        if (themeOrder[currentThemeIndex] === 'system') {
            // Theme variables update automatically via CSS
        }
    });

    applyTheme();

    // ─── TOC ──────────────────────────────────────────────────
    function buildTOC(tocData) {
        tocList.textContent = '';
        if (!tocData || tocData.length === 0) return;

        tocData.forEach(function(entry) {
            var li = document.createElement('li');
            var a = document.createElement('a');
            a.href = '#' + entry.id;
            a.textContent = entry.text;
            a.className = 'toc-h' + entry.level;
            a.dataset.id = entry.id;
            a.addEventListener('click', function(e) {
                e.preventDefault();
                var target = document.getElementById(entry.id);
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                    flashHighlight(target);
                }
            });
            li.appendChild(a);
            tocList.appendChild(li);
        });
    }

    function toggleTOC() {
        if (!config.toc || config.toc.length === 0) return;
        tocSidebar.classList.toggle('hidden');
        contentArea.classList.toggle('toc-open');
    }

    // Show TOC if --toc flag was set
    if (config.showTOC && config.toc && config.toc.length > 0) {
        tocSidebar.classList.remove('hidden');
        contentArea.classList.add('toc-open');
    }
    buildTOC(config.toc);

    // ─── Scroll Spy ───────────────────────────────────────────
    var scrollSpyObserver = null;

    function setupScrollSpy() {
        if (scrollSpyObserver) scrollSpyObserver.disconnect();

        var tocLinks = tocList.querySelectorAll('a');
        if (tocLinks.length === 0) return;

        scrollSpyObserver = new IntersectionObserver(function(entries) {
            entries.forEach(function(entry) {
                if (entry.isIntersecting) {
                    tocLinks.forEach(function(link) {
                        link.classList.remove('active');
                    });
                    var activeLink = tocList.querySelector('a[data-id="' + entry.target.id + '"]');
                    if (activeLink) {
                        activeLink.classList.add('active');
                        activeLink.scrollIntoView({ block: 'nearest' });
                    }
                }
            });
        }, {
            root: contentArea,
            rootMargin: '-40px 0px -80% 0px',
            threshold: 0
        });

        content.querySelectorAll('h1[id], h2[id], h3[id], h4[id], h5[id], h6[id]').forEach(function(h) {
            scrollSpyObserver.observe(h);
        });
    }

    setupScrollSpy();

    // ─── Heading Collection ───────────────────────────────────
    function collectHeadings() {
        headings = Array.from(content.querySelectorAll('h1, h2, h3, h4, h5, h6'));
    }

    collectHeadings();

    // ─── Heading Navigation ───────────────────────────────────
    // Returns true if navigation happened, false if at edge
    function navigateHeading(direction) {
        if (headings.length === 0) return false;

        var scrollY = contentArea.scrollTop + 50;

        var target = null;
        if (direction === 'next') {
            for (var i = 0; i < headings.length; i++) {
                if (headings[i].offsetTop > scrollY) {
                    target = headings[i];
                    break;
                }
            }
        } else {
            for (var j = headings.length - 1; j >= 0; j--) {
                if (headings[j].offsetTop < scrollY) {
                    target = headings[j];
                    break;
                }
            }
        }

        if (target) {
            target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            flashHighlight(target);
            return true;
        }
        return false; // at edge — no more headings in this direction
    }

    function flashHighlight(el) {
        el.classList.add('heading-flash');
        setTimeout(function() {
            el.classList.remove('heading-flash');
        }, 600);
    }

    // ─── Status Bar ───────────────────────────────────────────
    function updateStatusInfo(wordCount) {
        var readTime = Math.max(1, Math.round(wordCount / 250));
        statusInfo.textContent = wordCount.toLocaleString() + ' words \u00B7 ' + readTime + ' min read';
    }

    function updateScrollPercent() {
        var scrollTop = contentArea.scrollTop;
        var scrollHeight = contentArea.scrollHeight - contentArea.clientHeight;
        if (scrollHeight <= 0) {
            statusScroll.textContent = '0%';
            return;
        }
        var pct = Math.round((scrollTop / scrollHeight) * 100);
        statusScroll.textContent = pct + '%';
    }

    contentArea.addEventListener('scroll', updateScrollPercent);
    updateStatusInfo(config.wordCount || 0);
    updateScrollPercent();

    // ─── Code Block Copy Buttons ──────────────────────────────
    function makeCopyIcon() {
        var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('viewBox', '0 0 16 16');
        svg.setAttribute('fill', 'currentColor');
        var p1 = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        p1.setAttribute('d', 'M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25v-7.5z');
        var p2 = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        p2.setAttribute('d', 'M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25v-7.5zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25h-7.5z');
        svg.appendChild(p1);
        svg.appendChild(p2);
        return svg;
    }

    function makeCheckIcon() {
        var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('viewBox', '0 0 16 16');
        svg.setAttribute('fill', 'currentColor');
        var p = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        p.setAttribute('d', 'M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.75.75 0 1 1 1.06-1.06L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0z');
        svg.appendChild(p);
        return svg;
    }

    function createCopyBtn(codeEl) {
        var btn = document.createElement('button');
        btn.className = 'code-copy-btn';
        btn.title = 'Copy';
        btn.appendChild(makeCopyIcon());
        btn.addEventListener('click', function() {
            navigator.clipboard.writeText(codeEl.textContent).then(function() {
                btn.textContent = '';
                btn.appendChild(makeCheckIcon());
                btn.classList.add('visible');
                setTimeout(function() {
                    btn.textContent = '';
                    btn.appendChild(makeCopyIcon());
                    btn.classList.remove('visible');
                }, 2000);
            });
        });
        return btn;
    }

    function addCopyButtons() {
        document.querySelectorAll('.code-block-wrapper').forEach(function(wrapper) {
            if (wrapper.querySelector('.code-block-header')) return;
            var code = wrapper.querySelector('code');
            if (!code) return;

            var header = document.createElement('div');
            header.className = 'code-block-header';

            // Move existing language label into header
            var existingLabel = wrapper.querySelector('.code-lang-label');
            if (existingLabel) {
                header.appendChild(existingLabel);
            }

            header.appendChild(createCopyBtn(code));
            wrapper.appendChild(header);
        });

        // Also add to bare pre blocks (no wrapper)
        document.querySelectorAll('.markdown-body > pre').forEach(function(pre) {
            if (pre.parentElement.classList.contains('code-block-wrapper')) return;
            if (pre.querySelector('.code-copy-btn')) return;
            pre.style.position = 'relative';
            var code = pre.querySelector('code') || pre;

            var header = document.createElement('div');
            header.className = 'code-block-header';
            header.appendChild(createCopyBtn(code));
            pre.appendChild(header);
        });
    }

    addCopyButtons();

    // ─── Search ───────────────────────────────────────────────
    function openSearch() {
        searchBar.classList.remove('hidden');
        searchInput.focus();
        searchInput.select();
    }

    function closeSearch() {
        searchBar.classList.add('hidden');
        searchInput.value = '';
        clearHighlights();
        searchMatches = [];
        searchIndex = -1;
        searchCount.textContent = '';
    }

    function performSearch(query) {
        clearHighlights();
        searchMatches = [];
        searchIndex = -1;

        if (!query) {
            searchCount.textContent = '';
            return;
        }

        var lowerQuery = query.toLowerCase();

        // Walk text nodes in #content
        var walker = document.createTreeWalker(content, NodeFilter.SHOW_TEXT, null, false);
        var textNodes = [];
        var node;
        while ((node = walker.nextNode())) {
            // Skip nodes inside KaTeX
            if (node.parentElement && node.parentElement.closest('.katex')) continue;

            if (node.nodeValue && node.nodeValue.toLowerCase().indexOf(lowerQuery) !== -1) {
                textNodes.push(node);
            }
        }

        if (textNodes.length === 0) {
            searchCount.textContent = '0 results';
            return;
        }

        // Mark matches
        textNodes.forEach(function(tn) {
            markTextNode(tn, query);
        });

        // Collect marked elements
        var marks = document.querySelectorAll('mark.search-mark');
        marks.forEach(function(mark) {
            searchMatches.push(mark);
        });

        if (searchMatches.length > 0) {
            searchIndex = 0;
            highlightActive();
        }
        updateSearchCount();
    }

    function markTextNode(tn, query) {
        var text = tn.nodeValue;
        var lowerText = text.toLowerCase();
        var lowerQuery = query.toLowerCase();
        var idx = lowerText.indexOf(lowerQuery);
        if (idx === -1) return;

        // SVG text nodes: use overlay divs instead of <mark> insertion
        if (tn.parentElement && tn.parentElement.closest('svg')) {
            var svgEl = tn.parentElement.closest('svg');
            var mermaidContainer = svgEl.closest('.mermaid-rendered');
            if (!mermaidContainer) return;
            mermaidContainer.style.position = 'relative';

            // Find all occurrences and create overlay for each
            var searchIdx = 0;
            while (searchIdx < lowerText.length) {
                var pos = lowerText.indexOf(lowerQuery, searchIdx);
                if (pos === -1) break;

                // Use range to get position of the match
                var parentEl = tn.parentElement;
                if (parentEl && typeof parentEl.getBoundingClientRect === 'function') {
                    var containerRect = mermaidContainer.getBoundingClientRect();
                    var parentRect = parentEl.getBoundingClientRect();

                    var overlay = document.createElement('div');
                    overlay.className = 'svg-search-mark search-mark';
                    overlay.style.left = (parentRect.left - containerRect.left) + 'px';
                    overlay.style.top = (parentRect.top - containerRect.top) + 'px';
                    overlay.style.width = parentRect.width + 'px';
                    overlay.style.height = parentRect.height + 'px';
                    mermaidContainer.appendChild(overlay);
                }
                searchIdx = pos + lowerQuery.length;
            }
            return;
        }

        var parent = tn.parentNode;
        var before = text.substring(0, idx);
        var match = text.substring(idx, idx + query.length);
        var after = text.substring(idx + query.length);

        var mark = document.createElement('mark');
        mark.className = 'search-mark';
        mark.textContent = match;

        if (before) parent.insertBefore(document.createTextNode(before), tn);
        parent.insertBefore(mark, tn);
        if (after) {
            var afterNode = document.createTextNode(after);
            parent.insertBefore(afterNode, tn);
            // Recursively mark remaining matches in the after text
            markTextNode(afterNode, query);
        }
        parent.removeChild(tn);
    }

    function highlightActive() {
        searchMatches.forEach(function(el) {
            el.classList.remove('search-active-mark');
        });
        if (searchIndex >= 0 && searchIndex < searchMatches.length) {
            var active = searchMatches[searchIndex];
            active.classList.add('search-active-mark');
            active.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }

    function nextMatch() {
        if (searchMatches.length === 0) return;
        searchIndex = (searchIndex + 1) % searchMatches.length;
        highlightActive();
        updateSearchCount();
    }

    function prevMatch() {
        if (searchMatches.length === 0) return;
        searchIndex = (searchIndex - 1 + searchMatches.length) % searchMatches.length;
        highlightActive();
        updateSearchCount();
    }

    function updateSearchCount() {
        if (searchMatches.length === 0) {
            searchCount.textContent = '0 results';
        } else {
            searchCount.textContent = (searchIndex + 1) + ' of ' + searchMatches.length;
        }
    }

    function clearHighlights() {
        document.querySelectorAll('mark.search-mark').forEach(function(mark) {
            var parent = mark.parentNode;
            parent.replaceChild(document.createTextNode(mark.textContent), mark);
            parent.normalize();
        });
        // Remove SVG overlay marks
        document.querySelectorAll('.svg-search-mark').forEach(function(el) {
            el.remove();
        });
    }

    // Search event listeners
    searchInput.addEventListener('input', function() {
        performSearch(searchInput.value);
    });

    searchInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (e.shiftKey) { prevMatch(); } else { nextMatch(); }
        }
    });

    document.getElementById('search-next').addEventListener('click', nextMatch);
    document.getElementById('search-prev').addEventListener('click', prevMatch);
    document.getElementById('search-close').addEventListener('click', closeSearch);
    document.getElementById('search-btn').addEventListener('click', openSearch);

    // ─── Toolbar Buttons ──────────────────────────────────────
    document.getElementById('theme-toggle').addEventListener('click', cycleTheme);
    document.getElementById('close-btn').addEventListener('click', function() {
        if (typeof window.closeThisWindow === 'function') {
            window.closeThisWindow();
        } else {
            fetch('/api/shutdown', { method: 'POST' }).catch(function() {});
        }
    });

    // ─── Window Drag ──────────────────────────────────────────
    var winDragging = false;
    var winStartScreenX = 0;
    var winStartScreenY = 0;

    dragRegion.addEventListener('mousedown', function(e) {
        if (e.target.closest('button, input')) return;
        if (typeof window.moveWindowBy === 'function') {
            winDragging = true;
            winStartScreenX = e.screenX;
            winStartScreenY = e.screenY;
            e.preventDefault();
        }
    });

    document.addEventListener('mousemove', function(e) {
        if (!winDragging) return;
        var dx = e.screenX - winStartScreenX;
        var dy = e.screenY - winStartScreenY;
        winStartScreenX = e.screenX;
        winStartScreenY = e.screenY;
        window.moveWindowBy(dx, dy);
    });

    document.addEventListener('mouseup', function() {
        winDragging = false;
    });

    // ─── Zoom ─────────────────────────────────────────────────
    function applyZoom() {
        document.body.className = document.body.className.replace(/zoom-\d+/g, '').trim();
        document.body.classList.add('zoom-' + zoomLevels[zoomIndex]);
    }

    function zoomIn() {
        if (zoomIndex < zoomLevels.length - 1) {
            zoomIndex++;
            applyZoom();
        }
    }

    function zoomOut() {
        if (zoomIndex > 0) {
            zoomIndex--;
            applyZoom();
        }
    }

    function zoomReset() {
        zoomIndex = 2;
        applyZoom();
    }

    // ─── Keyboard Shortcuts ───────────────────────────────────
    document.addEventListener('keydown', function(e) {
        // Let search/mermaid inputs handle their own keys
        if (e.target === searchInput && e.key !== 'Escape') return;

        // Dismiss help overlay on any key
        if (!helpOverlay.classList.contains('hidden')) {
            helpOverlay.classList.add('hidden');
            return;
        }

        // Cmd/Ctrl+F
        if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
            e.preventDefault();
            openSearch();
            return;
        }

        // Escape
        if (e.key === 'Escape') {
            e.preventDefault();
            if (!mermaidModal.classList.contains('hidden')) {
                closeMermaidModal();
            } else if (!searchBar.classList.contains('hidden')) {
                closeSearch();
            } else {
                if (typeof window.closeThisWindow === 'function') {
                    window.closeThisWindow();
                } else {
                    fetch('/api/shutdown', { method: 'POST' }).catch(function() {});
                }
            }
            return;
        }

        // Don't handle shortcuts when typing in inputs
        if (e.target.matches('input, textarea, [contenteditable]')) return;

        var key = e.key;

        // j/k scroll
        if (key === 'j') { e.preventDefault(); contentArea.scrollBy({ top: 60, behavior: 'smooth' }); return; }
        if (key === 'k') { e.preventDefault(); contentArea.scrollBy({ top: -60, behavior: 'smooth' }); return; }

        // Space / Shift+Space page scroll
        if (key === ' ') {
            e.preventDefault();
            var delta = e.shiftKey ? -(contentArea.clientHeight - 60) : (contentArea.clientHeight - 60);
            contentArea.scrollBy({ top: delta, behavior: 'smooth' });
            return;
        }

        // g g top — scroll to absolute 0 to show full top padding
        if (key === 'g' && !e.shiftKey) {
            e.preventDefault();
            if (lastKeyWasG) {
                contentArea.scrollTop = 0;
                lastKeyWasG = false;
                clearTimeout(gKeyTimer);
                return;
            }
            lastKeyWasG = true;
            gKeyTimer = setTimeout(function() { lastKeyWasG = false; }, 500);
            return;
        } else {
            lastKeyWasG = false;
        }

        // G bottom
        if (key === 'G') { e.preventDefault(); contentArea.scrollTo({ top: contentArea.scrollHeight, behavior: 'smooth' }); return; }

        // Heading navigation — no wrapping at edges
        if (key === 'n' && !e.shiftKey) { e.preventDefault(); navigateHeading('next'); return; }
        if (key === 'p' && !e.shiftKey) { e.preventDefault(); navigateHeading('prev'); return; }

        // TOC toggle
        if (key === ']') { e.preventDefault(); toggleTOC(); return; }

        // Search
        if (key === '/') { e.preventDefault(); openSearch(); return; }

        // Theme
        if (key === 'T') { e.preventDefault(); cycleTheme(); return; }

        // Zoom
        if (key === '+' || key === '=') { e.preventDefault(); zoomIn(); return; }
        if (key === '-') { e.preventDefault(); zoomOut(); return; }
        if (key === '0') { e.preventDefault(); zoomReset(); return; }

        // Help
        if (key === 'h') { e.preventDefault(); helpOverlay.classList.toggle('hidden'); return; }
    });

    // ─── Lazy Loading: KaTeX ──────────────────────────────────
    function loadKaTeX() {
        if (!config.hasMath) return Promise.resolve();

        return new Promise(function(resolve) {
            var link = document.createElement('link');
            link.rel = 'stylesheet';
            link.href = '/static/vendor/katex.min.css';
            document.head.appendChild(link);

            var script = document.createElement('script');
            script.src = '/static/vendor/katex.min.js';
            script.onload = function() {
                renderMath();
                resolve();
            };
            script.onerror = function() { resolve(); };
            document.head.appendChild(script);
        });
    }

    function renderMath() {
        if (typeof katex === 'undefined') return;

        document.querySelectorAll('.math-block').forEach(function(el) {
            try {
                katex.render(el.textContent, el, { displayMode: true, throwOnError: false, trust: true });
            } catch (e) { /* leave as-is */ }
        });

        document.querySelectorAll('.math-inline').forEach(function(el) {
            try {
                katex.render(el.textContent, el, { displayMode: false, throwOnError: false, trust: true });
            } catch (e) { /* leave as-is */ }
        });
    }

    // ─── Lazy Loading: Mermaid ────────────────────────────────
    function loadMermaid() {
        if (!config.hasMermaid) return Promise.resolve();

        return new Promise(function(resolve) {
            var script = document.createElement('script');
            script.src = '/static/vendor/mermaid.min.js';
            script.onload = function() {
                renderMermaidDiagrams().then(function() {
                    attachMermaidClickHandlers();
                    resolve();
                });
            };
            script.onerror = function() { resolve(); };
            document.head.appendChild(script);
        });
    }

    function getEffectiveTheme() {
        var theme = themeOrder[currentThemeIndex];
        if (theme === 'system') {
            return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        }
        return theme;
    }

    async function renderMermaidDiagrams() {
        if (typeof mermaid === 'undefined') return;

        var effective = getEffectiveTheme();
        mermaid.initialize({
            startOnLoad: false,
            theme: effective === 'dark' ? 'dark' : 'default',
            themeVariables: {
                background: 'transparent'
            },
            securityLevel: 'strict',
            logLevel: 'error'
        });

        var placeholders = document.querySelectorAll('.mermaid-placeholder');
        for (var i = 0; i < placeholders.length; i++) {
            var el = placeholders[i];
            var source = el.textContent.trim();
            // Decode HTML entities
            var temp = document.createElement('textarea');
            temp.textContent = source;
            source = temp.value;

            try {
                var id = 'mermaid-' + i + '-' + Date.now();
                var result = await mermaid.render(id, source);
                el.textContent = '';
                el.insertAdjacentHTML('afterbegin', result.svg);
                el.classList.add('mermaid-rendered');
            } catch (err) {
                el.textContent = 'Mermaid error: ' + err.message;
                el.style.color = 'var(--admonition-caution)';
            }
        }
    }

    // ─── WebSocket Live Reload ────────────────────────────────
    function connectWS() {
        var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(proto + '//' + location.host + '/ws');

        ws.onopen = function() {
            disconnectedBanner.classList.add('hidden');
            reconnectDelay = 500;
        };

        ws.onmessage = function(event) {
            try {
                var msg = JSON.parse(event.data);
                if (msg.type === 'update') {
                    handleUpdate(msg);
                }
            } catch (e) {
                // Ignore parse errors
            }
        };

        ws.onclose = function() {
            disconnectedBanner.classList.remove('hidden');
            ws = null;
            setTimeout(function() {
                reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY);
                connectWS();
            }, reconnectDelay);
        };

        ws.onerror = function() {
            if (ws) ws.close();
        };
    }

    function handleUpdate(msg) {
        // Save scroll anchor (nearest heading)
        var scrollAnchor = findNearestHeading();

        // Update content using safe DOM method
        var tempDiv = document.createElement('div');
        tempDiv.insertAdjacentHTML('afterbegin', msg.html);
        content.textContent = '';
        while (tempDiv.firstChild) {
            content.appendChild(tempDiv.firstChild);
        }

        // Update config
        config.hasMath = msg.hasMath;
        config.hasMermaid = msg.hasMermaid;
        config.wordCount = msg.wordCount;
        config.toc = msg.toc;

        // Rebuild TOC
        buildTOC(msg.toc);

        // Re-collect headings
        collectHeadings();

        // Re-setup scroll spy
        setupScrollSpy();

        // Add copy buttons
        addCopyButtons();

        // Update status
        updateStatusInfo(msg.wordCount);

        // Restore scroll position
        restoreScrollAnchor(scrollAnchor);

        // Re-render math/mermaid if present
        if (config.hasMath) renderMath();
        if (config.hasMermaid) {
            renderMermaidDiagrams().then(attachMermaidClickHandlers);
        }

        // Re-run search if active
        if (!searchBar.classList.contains('hidden') && searchInput.value) {
            performSearch(searchInput.value);
        }
    }

    function findNearestHeading() {
        var scrollTop = contentArea.scrollTop;
        var nearest = null;
        var nearestDist = Infinity;
        headings.forEach(function(h) {
            var dist = Math.abs(h.offsetTop - scrollTop);
            if (dist < nearestDist) {
                nearestDist = dist;
                nearest = h.id;
            }
        });
        return nearest;
    }

    function restoreScrollAnchor(headingId) {
        if (!headingId) return;
        var el = document.getElementById(headingId);
        if (el) {
            contentArea.scrollTop = el.offsetTop - 50;
        }
    }

    // ─── Mermaid Modal (zoom/pan/search) ───────────────────────
    var mermaidModal = document.getElementById('mermaid-modal');
    var mermaidViewport = document.getElementById('mermaid-viewport');
    var mmZoomLevel = document.getElementById('mm-zoom-level');
    var mmZoom = 1;
    var mmPanX = 0, mmPanY = 0;
    var mmDragging = false, mmDragStartX = 0, mmDragStartY = 0;

    function openMermaidModal(svgContainer) {
        var svg = svgContainer.querySelector('svg');
        if (!svg) return;
        mermaidViewport.textContent = '';
        var clone = svg.cloneNode(true);
        clone.removeAttribute('style');
        clone.style.visibility = 'hidden';
        mermaidViewport.appendChild(clone);

        mmZoom = 1;
        mmPanX = 0;
        mmPanY = 0;
        mermaidModal.classList.remove('hidden');

        // Double rAF to ensure modal layout is complete before fitting
        requestAnimationFrame(function() {
            requestAnimationFrame(function() {
                mmFitToView();
                var fitted = mermaidViewport.querySelector('svg');
                if (fitted) fitted.style.visibility = 'visible';
            });
        });
    }

    function closeMermaidModal() {
        mermaidModal.classList.add('hidden');
        mermaidViewport.textContent = '';
    }

    function mmUpdateTransform() {
        var svg = mermaidViewport.querySelector('svg');
        if (!svg) return;
        svg.style.transform = 'translate(' + mmPanX + 'px, ' + mmPanY + 'px) scale(' + mmZoom + ')';
        mmZoomLevel.textContent = Math.round(mmZoom * 100) + '%';
    }

    function mmSetZoom(newZoom, centerX, centerY) {
        var vp = mermaidViewport;
        var cx = typeof centerX === 'number' ? centerX : vp.clientWidth / 2;
        var cy = typeof centerY === 'number' ? centerY : vp.clientHeight / 2;
        var ratio = newZoom / mmZoom;
        mmPanX = cx - ratio * (cx - mmPanX);
        mmPanY = cy - ratio * (cy - mmPanY);
        mmZoom = Math.max(0.1, Math.min(10, newZoom));
        mmUpdateTransform();
    }

    function mmFitToView() {
        var svg = mermaidViewport.querySelector('svg');
        if (!svg) return;

        // Reset transform to measure true content size (mermaid-preview-cli pattern)
        var prevTransform = svg.style.transform;
        svg.style.transform = 'none';

        var containerRect = mermaidViewport.getBoundingClientRect();
        var contentRect = svg.getBoundingClientRect();

        if (contentRect.width <= 0 || contentRect.height <= 0) {
            svg.style.transform = prevTransform;
            return;
        }

        var PADDING = 48;
        var scaleX = (containerRect.width - PADDING) / contentRect.width;
        var scaleY = (containerRect.height - PADDING) / contentRect.height;
        var fitZoom = Math.min(scaleX, scaleY);

        // Center: natural center of content relative to container
        var naturalCX = (contentRect.left - containerRect.left) + contentRect.width / 2;
        var naturalCY = (contentRect.top - containerRect.top) + contentRect.height / 2;

        mmZoom = fitZoom;
        mmPanX = containerRect.width / 2 - naturalCX * fitZoom;
        mmPanY = containerRect.height / 2 - naturalCY * fitZoom;
        mmUpdateTransform();
    }

    // Zoom controls
    document.getElementById('mm-zoom-in').addEventListener('click', function() { mmSetZoom(mmZoom * 1.25); });
    document.getElementById('mm-zoom-out').addEventListener('click', function() { mmSetZoom(mmZoom / 1.25); });
    document.getElementById('mm-zoom-fit').addEventListener('click', mmFitToView);
    document.getElementById('mm-zoom-reset').addEventListener('click', function() { mmSetZoom(1); });
    document.getElementById('mm-close').addEventListener('click', closeMermaidModal);

    // Mouse wheel zoom
    mermaidViewport.addEventListener('wheel', function(e) {
        e.preventDefault();
        var rect = mermaidViewport.getBoundingClientRect();
        var cx = e.clientX - rect.left;
        var cy = e.clientY - rect.top;
        var factor = e.deltaY < 0 ? 1.1 : 0.9;
        mmSetZoom(mmZoom * factor, cx, cy);
    }, { passive: false });

    // Pan by drag
    mermaidViewport.addEventListener('mousedown', function(e) {
        mmDragging = true;
        mmDragStartX = e.clientX - mmPanX;
        mmDragStartY = e.clientY - mmPanY;
        e.preventDefault();
    });

    document.addEventListener('mousemove', function(e) {
        if (!mmDragging) return;
        mmPanX = e.clientX - mmDragStartX;
        mmPanY = e.clientY - mmDragStartY;
        mmUpdateTransform();
    });

    document.addEventListener('mouseup', function() {
        mmDragging = false;
    });

    // Click handlers on rendered mermaid diagrams
    function attachMermaidClickHandlers() {
        document.querySelectorAll('.mermaid-rendered').forEach(function(el) {
            if (el.dataset.modalBound) return;
            el.dataset.modalBound = 'true';
            el.addEventListener('click', function() {
                openMermaidModal(el);
            });
        });
    }

    attachMermaidClickHandlers();

    // ─── Init ─────────────────────────────────────────────────
    async function init() {
        // Lazy-load KaTeX and Mermaid if needed
        await Promise.all([loadKaTeX(), loadMermaid()]);

        // Show window — 1:1.2 aspect ratio
        if (typeof window.showWindow === 'function') {
            var w = Math.min(1100, Math.round(screen.availWidth * 0.85));
            var h = Math.round(w * 1.2);
            if (h > screen.availHeight * 0.9) {
                h = Math.round(screen.availHeight * 0.9);
                w = Math.round(h / 1.2);
            }
            window.showWindow(w, h);
        }

        // Connect WebSocket for live reload
        if (!config.noWatch) {
            connectWS();
        }
    }

    init();
})();
