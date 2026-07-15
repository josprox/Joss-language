package files

import "path/filepath"

func GetAssetFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "assets", "js", "app.js"): `console.log('Joss Enterprise v3.0 - Inicializado');`,

		// SCSS Modules
		filepath.Join(path, "assets", "css", "_variables.scss"): `$primary: #2563eb;
$primary-dark: #1e40af;
$secondary: #64748b;
$background: #f8fafc;
$surface: #ffffff;
$text: #1e293b;
$text-light: #64748b;
$border: #e2e8f0;
$danger: #ef4444;
$success: #22c55e;
$info: #3b82f6;
`,
		filepath.Join(path, "assets", "css", "_layout.scss"): `body {
    font-family: 'Inter', sans-serif;
    background-color: $background;
    color: $text;
    margin: 0;
    line-height: 1.6;
    overflow-x: hidden;
}

.app-container {
    display: flex;
    min-height: 100vh;
    position: relative;
}

/* Sidebar */
.sidebar {
    width: 260px;
    background: $surface;
    border-right: 1px solid $border;
    display: flex;
    flex-direction: column;
    position: fixed;
    top: 0;
    left: 0;
    height: 100%;
    z-index: 1000;
    transition: transform 0.3s ease;
    
    @media (max-width: 768px) {
        transform: translateX(-100%);
        &.active {
            transform: translateX(0);
        }
    }
}

.sidebar-header {
    padding: 1.5rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid $border;
    
    .brand {
        font-size: 1.25rem;
        font-weight: 700;
        color: $primary;
        text-decoration: none;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
}

.sidebar-nav {
    flex: 1;
    overflow-y: auto;
    padding: 1rem 0;
    
    ul {
        list-style: none;
        padding: 0;
        margin: 0;
    }
    
    li {
        margin-bottom: 0.25rem;
    }
    
    a {
        display: flex;
        align-items: center;
        padding: 0.75rem 1.5rem;
        color: $text-light;
        text-decoration: none;
        font-weight: 500;
        transition: all 0.2s;
        gap: 0.75rem;
        
        &:hover, &.active {
            color: $primary;
            background: rgba($primary, 0.05);
            border-right: 3px solid $primary;
        }
        
        &.text-danger:hover {
            color: $danger;
            background: rgba($danger, 0.05);
            border-right-color: $danger;
        }
    }
    
    .nav-header {
        padding: 0.75rem 1.5rem;
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: #94a3b8;
        font-weight: 700;
        margin-top: 1rem;
    }
}

/* Main Content */
.main-content {
    flex: 1;
    margin-left: 260px;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    transition: margin-left 0.3s ease;
    
    @media (max-width: 768px) {
        margin-left: 0;
    }
}

/* Top Navbar */
.top-navbar {
    height: 64px;
    background: $surface;
    border-bottom: 1px solid $border;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 1.5rem;
    position: sticky;
    top: 0;
    z-index: 900;
}

.toggle-sidebar {
    background: none;
    border: none;
    font-size: 1.25rem;
    color: $text-light;
    cursor: pointer;
    display: none;
    
    @media (max-width: 768px) {
        display: block;
    }
}

.user-menu {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    
    .user-name {
        font-weight: 500;
        font-size: 0.9rem;
        @media (max-width: 576px) {
            display: none;
        }
    }
    
    .user-avatar {
        width: 36px;
        height: 36px;
        border-radius: 50%;
        object-fit: cover;
    }
}

/* Content Wrapper */
.content-wrapper {
    padding: 2rem;
    flex: 1;
    
    @media (max-width: 576px) {
        padding: 1rem;
    }
}

/* Footer */
.footer {
    padding: 1.5rem;
    text-align: center;
    color: $text-light;
    font-size: 0.9rem;
    border-top: 1px solid $border;
    background: $surface;
}

/* Overlay */
.sidebar-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0,0,0,0.5);
    z-index: 999;
    opacity: 0;
    visibility: hidden;
    transition: all 0.3s;
    
    &.active {
        opacity: 1;
        visibility: visible;
    }
}

/* Utilities */
.d-none { display: none !important; }
.d-sm-inline { @media (min-width: 576px) { display: inline !important; } }
.d-md-none { @media (min-width: 768px) { display: none !important; } }
.text-danger { color: $danger !important; }

.text-center { text-align: center; }
.d-flex { display: flex; }
.justify-content-center { justify-content: center; }
.justify-content-between { justify-content: space-between; }
.align-items-center { align-items: center; }
.gap-3 { gap: 1rem; }
.mt-4 { margin-top: 1.5rem; }
.mb-4 { margin-bottom: 1.5rem; }
.mb-5 { margin-bottom: 3rem; }
`,
		filepath.Join(path, "assets", "css", "_components.scss"): `.card {
    background: $surface;
    border-radius: 0.75rem;
    box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
    margin-bottom: 2rem;
    overflow: hidden;
}

.card-header {
    padding: 1.5rem;
    border-bottom: 1px solid $border;
    background: #f1f5f9;
}

.card-header h2 {
    margin: 0;
    font-size: 1.25rem;
}

.card-body {
    padding: 2rem;
}

.card-footer {
    padding: 1rem 2rem;
    background: #f8fafc;
    border-top: 1px solid $border;
}

.btn {
    display: inline-block;
    padding: 0.75rem 1.5rem;
    border-radius: 0.5rem;
    font-weight: 600;
    text-decoration: none;
    cursor: pointer;
    transition: all 0.2s;
    border: none;
}

.btn-primary {
    background-color: $primary;
    color: white;
}

.btn-primary:hover {
    background-color: $primary-dark;
}

.btn-outline-light {
    border: 2px solid $primary;
    color: $primary;
    background: transparent;
}

.btn-outline-light:hover {
    background: $primary;
    color: white;
}

.btn-outline-danger {
    border: 1px solid $danger;
    color: $danger;
    background: transparent;
    padding: 0.5rem 1rem;
}

.btn-outline-danger:hover {
    background: $danger;
    color: white;
}

.btn-block {
    display: block;
    width: 100%;
    text-align: center;
}

.form-group {
    margin-bottom: 1.5rem;
}

.form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
}

.form-control {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid $border;
    border-radius: 0.5rem;
    font-family: inherit;
    font-size: 1rem;
    box-sizing: border-box;
}

.form-control:focus {
    outline: none;
    border-color: $primary;
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.alert {
    padding: 1rem;
    border-radius: 0.5rem;
    margin-bottom: 1.5rem;
}

.alert-danger {
    background-color: #fef2f2;
    color: #991b1b;
    border: 1px solid #fecaca;
}

.alert-success {
    background-color: #f0fdf4;
    color: #166534;
    border: 1px solid #bbf7d0;
}

.alert-info {
    background-color: #eff6ff;
    color: #1e40af;
    border: 1px solid #dbeafe;
}

.badge {
    padding: 0.25rem 0.75rem;
    border-radius: 9999px;
    font-size: 0.875rem;
    font-weight: 600;
}

.badge-info {
    background-color: #e0f2fe;
    color: #0369a1;
}

.stat-card {
    background: #f8fafc;
    padding: 1.5rem;
    border-radius: 0.5rem;
    text-align: center;
    border: 1px solid $border;
}

.stat-number {
    font-size: 2.5rem;
    font-weight: 700;
    color: $primary;
    margin: 0.5rem 0 0;
}
`,
		filepath.Join(path, "assets", "css", "app.scss"): `// Main SCSS Entry Point
@import "variables";
@import "layout";
@import "components";
`,
	}
}
