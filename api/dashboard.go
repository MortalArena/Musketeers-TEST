package api

// DashboardHTML contains the embedded Single Page Application dashboard code
const DashboardHTML = `<!DOCTYPE html>
<html lang="ar" dir="rtl" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Musketeers Core Portal</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;700;800;900&family=Tajawal:wght@300;400;500;700;800;900&display=swap" rel="stylesheet">
    <style>
        :root[data-theme="dark"] {
            --bg-color: #030712;
            --bg-grid: rgba(255, 255, 255, 0.01);
            --panel-bg: rgba(17, 24, 39, 0.65);
            --panel-border: rgba(255, 255, 255, 0.08);
            --sidebar-bg: rgba(9, 13, 26, 0.9);
            --text-main: #f3f4f6;
            --text-muted: #9ca3af;
            --accent-cyan: #06b6d4;
            --accent-purple: #8b5cf6;
            --accent-emerald: #10b981;
            --accent-rose: #f43f5e;
            --accent-amber: #f59e0b;
            --shadow-glow: 0 0 25px rgba(6, 182, 212, 0.2);
            --card-hover: rgba(255, 255, 255, 0.03);
            --input-bg: rgba(15, 23, 42, 0.8);
            --chat-bubble-self: linear-gradient(135deg, rgba(6, 182, 212, 0.15), rgba(139, 92, 246, 0.15));
            --chat-bubble-other: rgba(31, 41, 55, 0.6);
            --topbar-bg: rgba(17, 24, 39, 0.4);
        }

        :root[data-theme="light"] {
            --bg-color: #f3f4f6;
            --bg-grid: rgba(0, 0, 0, 0.015);
            --panel-bg: rgba(255, 255, 255, 0.75);
            --panel-border: rgba(0, 0, 0, 0.08);
            --sidebar-bg: rgba(243, 244, 246, 0.95);
            --text-main: #1f2937;
            --text-muted: #4b5563;
            --accent-cyan: #0891b2;
            --accent-purple: #7c3aed;
            --accent-emerald: #059669;
            --accent-rose: #e11d48;
            --accent-amber: #d97706;
            --shadow-glow: 0 0 20px rgba(8, 145, 178, 0.15);
            --card-hover: rgba(0, 0, 0, 0.03);
            --input-bg: rgba(255, 255, 255, 0.95);
            --chat-bubble-self: linear-gradient(135deg, rgba(8, 145, 178, 0.1), rgba(124, 58, 237, 0.1));
            --chat-bubble-other: rgba(229, 231, 235, 0.7);
            --topbar-bg: rgba(243, 244, 246, 0.5);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
            transition: background-color 0.3s ease, border-color 0.3s ease;
        }

        body {
            background-color: var(--bg-color);
            color: var(--text-main);
            font-family: 'Tajawal', 'Outfit', sans-serif;
            min-height: 100vh;
            overflow: hidden;
            display: flex;
            background-image: 
                radial-gradient(at 0% 0%, rgba(139, 92, 246, 0.08) 0px, transparent 50%),
                radial-gradient(at 100% 100%, rgba(6, 182, 212, 0.08) 0px, transparent 50%);
        }

        html[dir="ltr"] {
            direction: ltr;
        }
        html[dir="rtl"] {
            direction: rtl;
        }

        /* App Layout Container */
        .app-container {
            display: flex;
            width: 100vw;
            height: 100vh;
            overflow: hidden;
        }

        /* Sidebar navigation */
        .sidebar {
            width: 280px;
            background: var(--sidebar-bg);
            border-inline-end: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            height: 100%;
            padding: 1.5rem 1.25rem;
            z-index: 100;
            flex-shrink: 0;
            backdrop-filter: blur(15px);
        }

        .logo-area {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            margin-bottom: 2rem;
            padding: 0.5rem;
        }

        .logo-icon {
            width: 40px;
            height: 40px;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 900;
            font-size: 1.3rem;
            color: #030712;
            box-shadow: var(--shadow-glow);
        }

        .logo-text {
            font-size: 1.3rem;
            font-weight: 800;
            background: linear-gradient(to right, var(--accent-cyan), var(--text-main));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .nav-menu {
            display: flex;
            flex-direction: column;
            gap: 0.35rem;
            flex-grow: 1;
            overflow-y: auto;
            padding-bottom: 1rem;
        }

        .nav-item {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.75rem 1rem;
            background: transparent;
            border: 1px solid transparent;
            border-radius: 10px;
            color: var(--text-muted);
            font-size: 0.95rem;
            font-weight: 700;
            cursor: pointer;
            width: 100%;
            transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
            text-align: start;
        }

        .nav-item:hover {
            color: var(--text-main);
            background: var(--card-hover);
        }

        .nav-item.active {
            color: #030712;
            background: linear-gradient(90deg, var(--accent-cyan), #ffffff);
            box-shadow: var(--shadow-glow);
            border-color: var(--accent-cyan);
        }

        html[dir="ltr"] .nav-item.active {
            background: linear-gradient(90deg, #ffffff, var(--accent-cyan));
        }

        .nav-item svg {
            width: 20px;
            height: 20px;
            stroke: var(--text-muted);
            fill: none;
            stroke-width: 2.2;
            transition: stroke 0.2s ease;
        }

        .nav-item.active svg {
            stroke: #030712;
        }

        .sidebar-footer {
            border-top: 1px solid var(--panel-border);
            padding-top: 1rem;
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
        }

        .footer-profile {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.5rem;
            border-radius: 10px;
            cursor: pointer;
            transition: background 0.2s ease;
        }

        .footer-profile:hover {
            background: var(--card-hover);
        }

        .avatar-circle {
            width: 38px;
            height: 38px;
            border-radius: 50%;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            box-shadow: 0 0 8px rgba(6, 182, 212, 0.25);
            border: 2px solid var(--panel-border);
            flex-shrink: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 800;
            color: #030712;
            font-size: 0.9rem;
        }

        .profile-info {
            display: flex;
            flex-direction: column;
            gap: 0.1rem;
            overflow: hidden;
        }

        .profile-name {
            font-weight: 700;
            font-size: 0.9rem;
            color: var(--text-main);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .profile-status {
            font-size: 0.75rem;
            color: var(--accent-emerald);
            display: flex;
            align-items: center;
            gap: 0.25rem;
        }

        /* Top Bar & Content Area */
        .workspace {
            display: flex;
            flex-direction: column;
            flex-grow: 1;
            height: 100%;
            overflow: hidden;
        }

        .top-bar {
            height: 70px;
            background: var(--topbar-bg);
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 2rem;
            backdrop-filter: blur(10px);
            z-index: 90;
            flex-shrink: 0;
        }

        /* Universal Search */
        .universal-search-container {
            position: relative;
            width: 400px;
        }

        .universal-search-bar {
            width: 100%;
            background: var(--input-bg);
            border: 1px solid var(--panel-border);
            border-radius: 20px;
            padding: 0.5rem 1rem;
            padding-inline-start: 2.5rem;
            color: var(--text-main);
            font-family: inherit;
            font-size: 0.9rem;
            transition: all 0.2s ease;
        }

        .universal-search-bar:focus {
            outline: none;
            border-color: var(--accent-cyan);
            box-shadow: 0 0 10px rgba(6, 182, 212, 0.15);
        }

        .search-icon-absolute {
            position: absolute;
            top: 50%;
            transform: translateY(-50%);
            left: 1rem;
            display: flex;
            align-items: center;
            pointer-events: none;
        }
        html[dir="rtl"] .search-icon-absolute {
            left: auto;
            right: 1rem;
        }

        .search-icon-absolute svg {
            width: 18px;
            height: 18px;
            stroke: var(--text-muted);
            fill: none;
            stroke-width: 2.2;
        }

        /* Autocomplete results dropdown */
        .autocomplete-dropdown {
            position: absolute;
            top: calc(100% + 5px);
            left: 0;
            right: 0;
            background: var(--sidebar-bg);
            border: 1px solid var(--panel-border);
            border-radius: 12px;
            max-height: 280px;
            overflow-y: auto;
            box-shadow: 0 10px 25px rgba(0,0,0,0.3);
            display: none;
            z-index: 150;
            backdrop-filter: blur(15px);
        }

        .autocomplete-item {
            padding: 0.75rem 1rem;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: space-between;
            border-bottom: 1px solid var(--panel-border);
        }

        .autocomplete-item:last-child {
            border-bottom: none;
        }

        .autocomplete-item:hover {
            background: var(--card-hover);
        }

        .autocomplete-title {
            font-weight: 700;
            font-size: 0.85rem;
            color: var(--text-main);
        }

        .autocomplete-type {
            font-size: 0.7rem;
            color: var(--accent-cyan);
            background: rgba(6, 182, 212, 0.1);
            padding: 0.15rem 0.5rem;
            border-radius: 8px;
            font-weight: 800;
        }

        /* Topbar Controls */
        .topbar-controls {
            display: flex;
            align-items: center;
            gap: 1.25rem;
        }

        /* Language Toggle */
        .lang-toggle-btn {
            background: transparent;
            border: 1px solid var(--panel-border);
            color: var(--text-main);
            padding: 0.35rem 0.75rem;
            border-radius: 8px;
            font-weight: 700;
            font-size: 0.8rem;
            cursor: pointer;
            font-family: inherit;
        }

        .lang-toggle-btn:hover {
            border-color: var(--text-muted);
        }

        /* Notifications Bell */
        .notif-bell-container {
            position: relative;
        }

        .notif-bell-btn {
            background: transparent;
            border: none;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            color: var(--text-muted);
            padding: 0.25rem;
            border-radius: 50%;
            transition: all 0.2s ease;
        }

        .notif-bell-btn:hover {
            color: var(--text-main);
            background: var(--card-hover);
        }

        .notif-bell-btn svg {
            width: 22px;
            height: 22px;
            stroke: currentColor;
            fill: none;
            stroke-width: 2.2;
        }

        .notif-counter {
            position: absolute;
            top: -2px;
            right: -2px;
            background: var(--accent-rose);
            color: white;
            font-size: 0.7rem;
            font-weight: 800;
            width: 16px;
            height: 16px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            border: 2px solid var(--bg-color);
        }

        .notif-dropdown {
            position: absolute;
            top: calc(100% + 15px);
            right: -10px;
            background: var(--sidebar-bg);
            border: 1px solid var(--panel-border);
            border-radius: 16px;
            width: 320px;
            box-shadow: 0 15px 35px rgba(0,0,0,0.3);
            display: none;
            z-index: 140;
            overflow: hidden;
            backdrop-filter: blur(15px);
        }
        html[dir="ltr"] .notif-dropdown {
            right: auto;
            left: -10px;
        }

        .notif-header {
            padding: 1rem;
            border-bottom: 1px solid var(--panel-border);
            font-weight: 800;
            font-size: 0.9rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .notif-clear-btn {
            font-size: 0.75rem;
            color: var(--accent-cyan);
            background: transparent;
            border: none;
            cursor: pointer;
            font-weight: 700;
        }

        .notif-list {
            max-height: 280px;
            overflow-y: auto;
        }

        .notif-item {
            padding: 0.85rem 1rem;
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
            cursor: pointer;
            transition: background 0.2s ease;
        }

        .notif-item:hover {
            background: var(--card-hover);
        }

        .notif-item:last-child {
            border-bottom: none;
        }

        .notif-text {
            font-size: 0.85rem;
            color: var(--text-main);
            line-height: 1.4;
        }

        .notif-time {
            font-size: 0.7rem;
            color: var(--text-muted);
        }

        /* Profile Menu Dropdown */
        .topbar-profile-container {
            position: relative;
        }

        .topbar-profile-btn {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            background: transparent;
            border: none;
            cursor: pointer;
            color: var(--text-main);
        }

        .topbar-profile-btn svg {
            width: 16px;
            height: 16px;
            stroke: var(--text-muted);
            fill: none;
            stroke-width: 2.5;
        }

        .profile-dropdown {
            position: absolute;
            top: calc(100% + 15px);
            right: 0;
            background: var(--sidebar-bg);
            border: 1px solid var(--panel-border);
            border-radius: 16px;
            width: 220px;
            box-shadow: 0 15px 30px rgba(0,0,0,0.3);
            display: none;
            z-index: 140;
            overflow: hidden;
            backdrop-filter: blur(15px);
        }
        html[dir="ltr"] .profile-dropdown {
            right: auto;
            left: 0;
        }

        .profile-dropdown-item {
            padding: 0.85rem 1.25rem;
            display: flex;
            align-items: center;
            gap: 0.75rem;
            color: var(--text-main);
            font-size: 0.85rem;
            font-weight: 700;
            cursor: pointer;
            border-bottom: 1px solid var(--panel-border);
            transition: background 0.2s ease;
        }

        .profile-dropdown-item:last-child {
            border-bottom: none;
            color: var(--accent-rose);
        }

        .profile-dropdown-item:hover {
            background: var(--card-hover);
        }

        .profile-dropdown-item svg {
            width: 18px;
            height: 18px;
            stroke: currentColor;
            fill: none;
            stroke-width: 2.2;
        }

        /* Main View Container */
        .view-content-area {
            flex-grow: 1;
            padding: 1.5rem 2rem;
            overflow-y: auto;
            height: calc(100% - 70px);
        }

        /* Common Page Header */
        .page-header {
            margin-bottom: 1.5rem;
        }

        .page-header h1 {
            font-size: 1.75rem;
            font-weight: 900;
            letter-spacing: -0.5px;
            background: linear-gradient(to right, var(--accent-cyan), var(--accent-purple));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 0.25rem;
        }

        .page-header p {
            color: var(--text-muted);
            font-size: 0.95rem;
        }

        /* Base Card Styling */
        .card {
            background: var(--panel-bg);
            border: 1px solid var(--panel-border);
            border-radius: 20px;
            padding: 1.5rem;
            backdrop-filter: blur(15px);
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
            margin-bottom: 1.5rem;
        }

        /* Form Controls */
        .form-group {
            margin-bottom: 1.25rem;
        }

        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-size: 0.85rem;
            font-weight: 800;
            color: var(--text-muted);
        }

        .form-input {
            width: 100%;
            background: var(--input-bg);
            border: 1px solid var(--panel-border);
            color: var(--text-main);
            padding: 0.75rem 1rem;
            border-radius: 12px;
            font-family: inherit;
            font-size: 0.9rem;
            transition: all 0.2s ease;
        }

        .form-input:focus {
            outline: none;
            border-color: var(--accent-cyan);
            box-shadow: 0 0 10px rgba(6, 182, 212, 0.1);
        }

        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 0.5rem;
            padding: 0.75rem 1.5rem;
            border-radius: 12px;
            font-family: inherit;
            font-size: 0.9rem;
            font-weight: 800;
            cursor: pointer;
            transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
            border: 1px solid transparent;
        }

        .btn-primary {
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            color: #030712;
            box-shadow: 0 4px 15px rgba(6, 182, 212, 0.2);
        }

        .btn-primary:hover {
            transform: translateY(-1px);
            box-shadow: 0 6px 20px rgba(6, 182, 212, 0.35);
        }

        .btn-secondary {
            background: rgba(255, 255, 255, 0.03);
            border-color: var(--panel-border);
            color: var(--text-main);
        }

        .btn-secondary:hover {
            background: rgba(255, 255, 255, 0.08);
            border-color: var(--text-muted);
        }

        .btn-danger {
            background: rgba(244, 63, 94, 0.15);
            border-color: rgba(244, 63, 94, 0.3);
            color: var(--accent-rose);
        }

        .btn-danger:hover {
            background: var(--accent-rose);
            color: white;
        }

        /* Grid utilities */
        .grid-2 {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1.5rem;
        }
        .grid-3 {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1.5rem;
        }

        @media(max-width: 900px) {
            .grid-2, .grid-3 {
                grid-template-columns: 1fr;
            }
        }

        /* 🏠 HOME TAB STYLING */
        .passport-card {
            background: linear-gradient(135deg, rgba(6, 182, 212, 0.05), rgba(139, 92, 246, 0.05));
            border: 2px solid rgba(6, 182, 212, 0.15);
            position: relative;
            overflow: hidden;
        }

        .passport-card::before {
            content: '';
            position: absolute;
            top: -50px;
            right: -50px;
            width: 150px;
            height: 150px;
            background: radial-gradient(circle, rgba(139, 92, 246, 0.15) 0%, transparent 70%);
            pointer-events: none;
        }

        .passport-avatar-zone {
            display: flex;
            align-items: center;
            gap: 1.25rem;
            margin-bottom: 1.5rem;
        }

        .passport-avatar {
            width: 70px;
            height: 70px;
            border-radius: 18px;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.8rem;
            font-weight: 900;
            color: #030712;
            box-shadow: 0 0 15px rgba(6, 182, 212, 0.25);
            border: 2px solid var(--panel-border);
        }

        .passport-details {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
            font-size: 0.85rem;
            border-top: 1px solid var(--panel-border);
            padding-top: 1rem;
        }

        .passport-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .passport-key {
            color: var(--text-muted);
            font-weight: 700;
        }

        .passport-val {
            font-weight: 800;
            color: var(--text-main);
            font-family: monospace;
            max-width: 160px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .stat-card-row {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1rem;
            margin-bottom: 1.5rem;
        }

        .stat-card {
            background: var(--panel-bg);
            border: 1px solid var(--panel-border);
            border-radius: 16px;
            padding: 1.25rem;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .stat-num {
            font-size: 2rem;
            font-weight: 900;
            font-family: 'Outfit', sans-serif;
            color: var(--accent-cyan);
        }

        .stat-label {
            font-size: 0.8rem;
            font-weight: 800;
            color: var(--text-muted);
        }

        .recent-actions-card {
            max-height: 250px;
            overflow-y: auto;
        }

        .recent-item {
            padding: 0.75rem;
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
            font-size: 0.85rem;
        }

        .recent-item:last-child {
            border-bottom: none;
        }

        /* 💬 CHATS TAB STYLING */
        .chat-main-container {
            display: flex;
            background: var(--panel-bg);
            border: 1px solid var(--panel-border);
            border-radius: 20px;
            height: calc(100vh - 180px);
            overflow: hidden;
            backdrop-filter: blur(15px);
        }

        .chat-view-sidebar {
            width: 280px;
            border-inline-end: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            background: rgba(9, 13, 26, 0.4);
            flex-shrink: 0;
        }

        .chat-sidebar-search {
            padding: 1rem;
            border-bottom: 1px solid var(--panel-border);
        }

        .chat-list-scroll {
            flex-grow: 1;
            overflow-y: auto;
            padding: 0.5rem;
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }

        .chat-item-card {
            padding: 0.75rem 1rem;
            border-radius: 12px;
            cursor: pointer;
            display: flex;
            align-items: center;
            gap: 0.75rem;
            transition: all 0.2s ease;
        }

        .chat-item-card:hover {
            background: var(--card-hover);
        }

        .chat-item-card.active {
            background: rgba(6, 182, 212, 0.08);
            border: 1px solid rgba(6, 182, 212, 0.15);
        }

        .chat-item-info {
            display: flex;
            flex-direction: column;
            gap: 0.15rem;
            flex-grow: 1;
            overflow: hidden;
        }

        .chat-item-name {
            font-weight: 700;
            font-size: 0.85rem;
            color: var(--text-main);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .chat-item-preview {
            font-size: 0.75rem;
            color: var(--text-muted);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .chat-content-pane {
            flex-grow: 1;
            display: flex;
            flex-direction: column;
            height: 100%;
            overflow: hidden;
            justify-content: space-between;
        }

        .chat-pane-header {
            padding: 1rem 1.5rem;
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
            background: rgba(9, 13, 26, 0.2);
        }

        .chat-pane-title {
            font-weight: 800;
            font-size: 1rem;
            color: var(--text-main);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .chat-header-actions {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .chat-messages-scroll {
            flex-grow: 1;
            overflow-y: auto;
            padding: 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 1rem;
            background: rgba(0,0,0,0.15);
        }

        .message-bubble {
            max-width: 65%;
            padding: 0.75rem 1rem;
            border-radius: 16px;
            font-size: 0.9rem;
            line-height: 1.5;
            position: relative;
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
            word-break: break-word;
        }

        .message-bubble.incoming {
            align-self: flex-start;
            background: var(--chat-bubble-other);
            border: 1px solid var(--panel-border);
            border-bottom-right-radius: 4px;
        }
        html[dir="ltr"] .message-bubble.incoming {
            border-bottom-right-radius: 16px;
            border-bottom-left-radius: 4px;
        }

        .message-bubble.outgoing {
            align-self: flex-end;
            background: var(--chat-bubble-self);
            border: 1px solid rgba(6, 182, 212, 0.2);
            border-bottom-left-radius: 4px;
        }
        html[dir="ltr"] .message-bubble.outgoing {
            border-bottom-left-radius: 16px;
            border-bottom-right-radius: 4px;
        }

        .msg-sender-lbl {
            font-size: 0.7rem;
            font-weight: 800;
            color: var(--accent-cyan);
            margin-bottom: 0.1rem;
        }

        .msg-actions-hover {
            position: absolute;
            top: -20px;
            left: 10px;
            background: var(--sidebar-bg);
            border: 1px solid var(--panel-border);
            border-radius: 8px;
            display: none;
            gap: 0.25rem;
            padding: 0.15rem;
            box-shadow: 0 4px 10px rgba(0,0,0,0.2);
        }

        .message-bubble:hover .msg-actions-hover {
            display: flex;
        }

        .msg-action-btn {
            background: transparent;
            border: none;
            cursor: pointer;
            color: var(--text-muted);
            padding: 0.15rem;
            border-radius: 4px;
            display: flex;
            align-items: center;
        }

        .msg-action-btn:hover {
            color: var(--text-main);
            background: var(--card-hover);
        }

        .msg-action-btn svg {
            width: 14px;
            height: 14px;
            stroke: currentColor;
            fill: none;
        }

        .chat-input-row {
            padding: 1rem 1.5rem;
            border-top: 1px solid var(--panel-border);
            background: rgba(9, 13, 26, 0.2);
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .chat-input-controls {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .chat-drag-zone {
            border: 1.5px dashed var(--panel-border);
            border-radius: 10px;
            padding: 0.4rem;
            text-align: center;
            font-size: 0.75rem;
            color: var(--text-muted);
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .chat-drag-zone:hover {
            border-color: var(--accent-cyan);
            color: var(--text-main);
        }

        .shared-files-tab {
            padding: 1rem;
            display: none;
            flex-direction: column;
            gap: 0.5rem;
            height: 100%;
            overflow-y: auto;
        }

        /* 📢 CHANNELS TAB STYLING */
        .channel-pane-sidebar {
            width: 280px;
            border-inline-end: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            background: rgba(9, 13, 26, 0.4);
            flex-shrink: 0;
        }

        .channel-split-tabs {
            display: flex;
            border-bottom: 1px solid var(--panel-border);
        }

        .channel-split-btn {
            flex-grow: 1;
            padding: 0.75rem;
            background: transparent;
            border: none;
            color: var(--text-muted);
            font-weight: 800;
            font-size: 0.8rem;
            cursor: pointer;
            text-align: center;
            border-bottom: 2px solid transparent;
        }

        .channel-split-btn.active {
            color: var(--accent-cyan);
            border-bottom-color: var(--accent-cyan);
        }

        /* 🖥️ BROWSER TAB STYLING */
        .browser-container {
            border: 1px solid var(--panel-border);
            border-radius: 20px;
            overflow: hidden;
            display: flex;
            flex-direction: column;
            background: rgba(9, 13, 26, 0.3);
            height: calc(100vh - 180px);
            box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
        }

        .browser-toolbar {
            background: var(--sidebar-bg);
            border-bottom: 1px solid var(--panel-border);
            padding: 0.75rem 1.25rem;
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .browser-nav-group {
            display: flex;
            align-items: center;
            gap: 0.35rem;
        }

        .browser-nav-btn {
            background: transparent;
            border: none;
            color: var(--text-muted);
            cursor: pointer;
            padding: 0.25rem;
            border-radius: 6px;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .browser-nav-btn:hover {
            color: var(--text-main);
            background: var(--card-hover);
        }

        .browser-nav-btn svg {
            width: 18px;
            height: 18px;
            stroke: currentColor;
            fill: none;
            stroke-width: 2.2;
        }

        .browser-address-container {
            flex-grow: 1;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            background: var(--input-bg);
            border: 1px solid var(--panel-border);
            border-radius: 20px;
            padding: 0.45rem 1rem;
        }

        .browser-address-input {
            background: transparent;
            border: none;
            color: var(--text-main);
            font-family: inherit;
            font-size: 0.85rem;
            width: 100%;
            outline: none;
        }

        .browser-badge {
            display: inline-flex;
            align-items: center;
            gap: 0.25rem;
            font-size: 0.7rem;
            font-weight: 800;
            padding: 0.15rem 0.5rem;
            border-radius: 8px;
        }

        .browser-badge.verified {
            background: rgba(16, 185, 129, 0.15);
            color: var(--accent-emerald);
            border: 1px solid rgba(16, 185, 129, 0.25);
        }

        .browser-badge.unverified {
            background: rgba(245, 158, 11, 0.15);
            color: var(--accent-amber);
            border: 1px solid rgba(245, 158, 11, 0.25);
        }

        .browser-badge svg {
            width: 12px;
            height: 12px;
            stroke: currentColor;
            fill: none;
        }

        .browser-viewport {
            flex-grow: 1;
            background: #ffffff;
            position: relative;
        }

        .browser-mock-home {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: var(--bg-color);
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 2rem;
            text-align: center;
            gap: 1.5rem;
            overflow-y: auto;
        }

        .browser-history-list {
            max-width: 500px;
            width: 100%;
            background: var(--panel-bg);
            border: 1px solid var(--panel-border);
            border-radius: 12px;
            max-height: 180px;
            overflow-y: auto;
            text-align: start;
        }

        .history-item {
            padding: 0.5rem 1rem;
            border-bottom: 1px solid var(--panel-border);
            cursor: pointer;
            font-size: 0.8rem;
            display: flex;
            justify-content: space-between;
        }

        .history-item:hover {
            background: var(--card-hover);
        }

        /* 🔎 EXPLORE TAB STYLING */
        .explore-agents-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
            gap: 1rem;
            margin-bottom: 1.5rem;
        }

        .explore-agent-card {
            background: var(--panel-bg);
            border: 1px solid var(--panel-border);
            border-radius: 16px;
            padding: 1.25rem;
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
            transition: all 0.2s ease;
        }

        .explore-agent-card:hover {
            border-color: var(--accent-cyan);
            transform: translateY(-2px);
        }

        .network-map-canvas {
            width: 100%;
            height: 300px;
            background: rgba(0,0,0,0.25);
            border-radius: 16px;
            border: 1px solid var(--panel-border);
            position: relative;
            overflow: hidden;
        }

        /* 📁 FILES TAB STYLING */
        .file-drop-area {
            border: 2px dashed var(--panel-border);
            border-radius: 20px;
            padding: 3rem 1.5rem;
            text-align: center;
            cursor: pointer;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 1rem;
            background: rgba(255,255,255,0.01);
            transition: all 0.3s ease;
        }

        .file-drop-area:hover {
            border-color: var(--accent-cyan);
            background: var(--card-hover);
        }

        .file-drop-area svg {
            width: 48px;
            height: 48px;
            stroke: var(--accent-cyan);
            fill: none;
            stroke-width: 1.5;
        }

        /* Tables */
        .table-container {
            overflow-x: auto;
            width: 100%;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            text-align: start;
        }

        th {
            padding: 0.85rem 1rem;
            font-weight: 800;
            color: var(--text-muted);
            border-bottom: 2px solid var(--panel-border);
            font-size: 0.8rem;
        }

        td {
            padding: 0.85rem 1rem;
            border-bottom: 1px solid var(--panel-border);
            font-size: 0.85rem;
            color: var(--text-main);
        }

        tr:hover td {
            background: var(--card-hover);
        }

        /* 🛠️ DEV TOOLS TAB STYLING */
        .dev-terminal-view {
            background: #000000;
            border: 1px solid var(--panel-border);
            border-radius: 12px;
            font-family: monospace;
            color: #22c55e;
            padding: 1.25rem;
            height: 250px;
            overflow-y: auto;
            font-size: 0.8rem;
            display: flex;
            flex-direction: column;
            gap: 0.35rem;
            box-shadow: inset 0 0 10px rgba(0,0,0,0.8);
        }

        .resource-chart-row {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1rem;
        }

        .resource-chart-bar-container {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .resource-bar-outer {
            width: 100%;
            height: 24px;
            background: rgba(255,255,255,0.05);
            border-radius: 12px;
            overflow: hidden;
            border: 1px solid var(--panel-border);
        }

        .resource-bar-inner {
            height: 100%;
            background: linear-gradient(90deg, var(--accent-cyan), var(--accent-purple));
            width: 0%;
            transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
        }

        /* Modal Dialog Base */
        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.7);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 200;
            backdrop-filter: blur(5px);
        }

        .modal-content {
            background: var(--sidebar-bg);
            border: 1px solid var(--panel-border);
            border-radius: 24px;
            width: 100%;
            max-width: 480px;
            padding: 2rem;
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
            box-shadow: 0 20px 50px rgba(0,0,0,0.5);
            animation: modalIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
        }

        @keyframes modalIn {
            from { opacity: 0; transform: translateY(20px) scale(0.95); }
            to { opacity: 1; transform: translateY(0) scale(1); }
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .modal-header h3 {
            font-size: 1.25rem;
            font-weight: 800;
        }

        .close-modal-btn {
            background: transparent;
            border: none;
            color: var(--text-muted);
            cursor: pointer;
        }

        .close-modal-btn:hover {
            color: var(--text-main);
        }

        /* Toast Container */
        .toast-container {
            position: fixed;
            bottom: 2rem;
            left: 2rem;
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
            z-index: 1000;
        }
        html[dir="ltr"] .toast-container {
            left: auto;
            right: 2rem;
        }

        .toast {
            background: var(--sidebar-bg);
            border: 1px solid var(--panel-border);
            border-radius: 12px;
            padding: 1rem 1.5rem;
            color: var(--text-main);
            box-shadow: 0 10px 35px rgba(0, 0, 0, 0.3);
            display: flex;
            align-items: center;
            gap: 0.75rem;
            transform: translateX(-120%);
            animation: toastIn 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
            min-width: 300px;
            font-weight: 700;
            font-size: 0.85rem;
            backdrop-filter: blur(10px);
        }
        html[dir="ltr"] .toast {
            transform: translateX(120%);
        }

        @keyframes toastIn {
            to { transform: translateX(0); }
        }

        .toast.success { border-inline-start: 5px solid var(--accent-emerald); }
        .toast.error { border-inline-start: 5px solid var(--accent-rose); }
        .toast.warning { border-inline-start: 5px solid var(--accent-amber); }

        /* Helpers */
        .flex-between { justify-content: space-between; align-items: center; display: flex; }
        .flex-row-gap { display: flex; gap: 0.5rem; align-items: center; }
        .tab-panel { display: none; height: 100%; }
        .tab-panel.active { display: block; }
        .hidden { display: none !important; }
        .glowing-circle { animation: glowPulse 2s infinite alternate; }

        @keyframes glowPulse {
            from { box-shadow: 0 0 5px rgba(6, 182, 212, 0.2); }
            to { box-shadow: 0 0 18px rgba(6, 182, 212, 0.5); }
        }
    </style>
</head>
<body>

    <div class="app-container">
        <!-- Sidebar Navigation -->
        <div class="sidebar">
            <div>
                <div class="logo-area">
                    <div class="logo-icon">NR</div>
                    <div class="logo-text">Musketeers</div>
                </div>
                
                <div class="nav-menu">
                    <button class="nav-item active" onclick="switchTab('home', this)" data-tr="tab_home">
                        <svg viewBox="0 0 24 24"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path><polyline points="9 22 9 12 15 12 15 22"></polyline></svg>
                        <span>الرئيسية</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('chats', this)" data-tr="tab_chats">
                        <svg viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>
                        <span>المحادثات</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('channels', this)" data-tr="tab_channels">
                        <svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>
                        <span>القنوات</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('browser', this)" data-tr="tab_browser">
                        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>
                        <span>المتصفح اللامركزي</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('explore', this)" data-tr="tab_explore">
                        <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
                        <span>استكشاف</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('files', this)" data-tr="tab_files">
                        <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg>
                        <span>الملفات</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('contacts', this)" data-tr="tab_contacts">
                        <svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle></svg>
                        <span>جهات الاتصال</span>
                    </button>
                    <button class="nav-item" onclick="switchTab('devtools', this)" data-tr="tab_devtools">
                        <svg viewBox="0 0 24 24"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline><line x1="12" y1="4" x2="12" y2="20"></line></svg>
                        <span>أدوات المطور</span>
                    </button>
                </div>
            </div>

            <div class="sidebar-footer">
                <div class="footer-profile" onclick="switchTab('home')">
                    <div class="avatar-circle" id="sb-avatar-letter">N</div>
                    <div class="profile-info">
                        <span class="profile-name" id="sb-profile-name">did:ia:...</span>
                        <span class="profile-status">
                            <span style="width: 7px; height: 7px; border-radius: 50%; background: var(--accent-emerald); display: inline-block;"></span>
                            <span data-tr="connected_status">متصل بالشبكة</span>
                        </span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Workspace content area -->
        <div class="workspace">
            <!-- Top Bar -->
            <div class="top-bar">
                <div class="universal-search-container">
                    <span class="search-icon-absolute">
                        <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
                    </span>
                    <input type="text" class="universal-search-bar" id="topbar-search-input" placeholder="ابحث في الشبكة عن نطاق، وكيل، قناة أو محتوى..." oninput="handleUniversalSearchInput()" onkeydown="if(event.key==='Enter') executeUniversalSearch()">
                    <div class="autocomplete-dropdown" id="search-autocomplete-box"></div>
                </div>

                <div class="topbar-controls">
                    <!-- Theme Toggle -->
                    <button class="lang-toggle-btn" onclick="toggleTheme()" id="theme-toggle-btn">Dark</button>

                    <!-- Language Switcher -->
                    <button class="lang-toggle-btn" onclick="toggleLanguage()" id="lang-toggle-btn">English</button>

                    <!-- Notifications Dropdown -->
                    <div class="notif-bell-container">
                        <button class="notif-bell-btn" onclick="toggleNotifDropdown()">
                            <svg viewBox="0 0 24 24"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg>
                            <span class="notif-counter" id="notif-count-badge">0</span>
                        </button>
                        <div class="notif-dropdown" id="notif-dropdown-menu">
                            <div class="notif-header">
                                <span data-tr="notifications_title">الإشعارات</span>
                                <button class="notif-clear-btn" onclick="clearNotifications()" data-tr="notif_clear">تحديد كمقروء</button>
                            </div>
                            <div class="notif-list" id="notif-items-list">
                                <!-- Dynamic notifications -->
                            </div>
                        </div>
                    </div>

                    <!-- Profile dropdown menu -->
                    <div class="topbar-profile-container">
                        <button class="topbar-profile-btn" onclick="toggleProfileDropdown()">
                            <div class="avatar-circle" style="width:30px; height:30px; font-size:0.75rem;" id="topbar-avatar-letter">N</div>
                            <svg viewBox="0 0 24 24"><polyline points="6 9 12 15 18 9"></polyline></svg>
                        </button>
                        <div class="profile-dropdown" id="profile-dropdown-menu">
                            <div class="profile-dropdown-item" onclick="switchTab('home')" data-tr="profile_view">
                                <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
                                <span>الملف الشخصي</span>
                            </div>
                            <div class="profile-dropdown-item" onclick="switchTab('devtools')" data-tr="settings">
                                <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
                                <span>الإعدادات</span>
                            </div>
                            <div class="profile-dropdown-item" onclick="logoutSession()" data-tr="logout">
                                <svg viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
                                <span>تسجيل الخروج</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Workspace view components -->
            <div class="view-content-area">

                <!-- 🏠 TAB: HOME -->
                <div id="home-panel" class="tab-panel active">
                    <div class="page-header">
                        <h1 data-tr="home_title">الموجز الرئيسي للشبكة</h1>
                        <p data-tr="home_subtitle">متابعة نشاط العقدة، إحصائيات النظام وإدارة الهوية اللامركزية</p>
                    </div>

                    <div class="grid-2">
                        <!-- Passport Card -->
                        <div class="card passport-card">
                            <div class="passport-avatar-zone">
                                <div class="passport-avatar" id="home-passport-avatar">N</div>
                                <div>
                                    <h3 style="font-size:1.15rem; font-weight:800;" id="home-passport-nickname">Musketeers Node</h3>
                                    <span style="font-size:0.75rem; color:var(--accent-cyan);" id="home-passport-status">Active Identity</span>
                                </div>
                            </div>
                            <div class="passport-details">
                                <div class="passport-row">
                                    <span class="passport-key">DID</span>
                                    <span class="passport-val" id="home-passport-did">did:ia:...</span>
                                </div>
                                <div class="passport-row">
                                    <span class="passport-key" data-tr="passport_created">تاريخ الإنشاء</span>
                                    <span class="passport-val" id="home-passport-created">...</span>
                                </div>
                                <div class="passport-row">
                                    <span class="passport-key" data-tr="passport_reputation">السمعة والوثوقية</span>
                                    <span class="passport-val" style="color:var(--accent-emerald);">98% (Excellent)</span>
                                </div>
                                <div class="passport-row">
                                    <span class="passport-key" data-tr="passport_domains">النطاقات النشطة</span>
                                    <span class="passport-val" id="home-passport-domain-count">0 Domains</span>
                                </div>
                            </div>
                            <div style="margin-top:1.25rem; display:flex; gap:0.5rem;">
                                <button class="btn btn-secondary" style="font-size:0.75rem; padding:0.5rem 1rem;" onclick="copyToClipboardText(document.getElementById('home-passport-did').innerText, 'تم نسخ الـ DID')" data-tr="copy_did">نسخ المعرف</button>
                            </div>
                        </div>

                        <!-- Statistics -->
                        <div>
                            <div class="stat-card-row">
                                <div class="stat-card">
                                    <span class="stat-num" id="home-stat-peers">0</span>
                                    <span class="stat-label" data-tr="peers_count">الوكلاء المتصلون</span>
                                </div>
                                <div class="stat-card">
                                    <span class="stat-num" id="home-stat-domains">0</span>
                                    <span class="stat-label" data-tr="registered_domains_count">النطاقات بالشبكة</span>
                                </div>
                                <div class="stat-card">
                                    <span class="stat-num">38.4 KB</span>
                                    <span class="stat-label" data-tr="local_storage_size">سعة التخزين المحلي</span>
                                </div>
                            </div>

                            <div class="card">
                                <h3 style="font-size:0.95rem; font-weight:800; margin-bottom:1rem;" data-tr="quick_actions">إجراءات سريعة</h3>
                                <div style="display:flex; flex-direction:column; gap:0.75rem;">
                                    <button class="btn btn-primary" onclick="switchTab('devtools')" data-tr="act_register">تسجيل نطاق جديد (.ia)</button>
                                    <div style="display:grid; grid-template-columns:1fr 1fr; gap:0.5rem;">
                                        <button class="btn btn-secondary" onclick="switchTab('files')" data-tr="act_upload">رفع ملف جديد</button>
                                        <button class="btn btn-secondary" onclick="switchTab('channels')" data-tr="act_channel">إنشاء قناة GossipSub</button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="recent_activities">آخر النشاطات العامة</h3>
                        <div class="recent-actions-card" id="home-activities-list">
                            <!-- Dynamic recent activities -->
                        </div>
                    </div>
                </div>

                <!-- 💬 TAB: CHATS -->
                <div id="chats-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="chats_title">محادثات الوكلاء المشفرة</h1>
                        <p data-tr="chats_subtitle">تواصل خاص وسري 1:1 مؤمن بالكامل عبر خوارزميات التشفير E2E</p>
                    </div>

                    <div class="chat-main-container">
                        <!-- Chats Sidebar -->
                        <div class="chat-view-sidebar">
                            <div class="chat-sidebar-search">
                                <input type="text" class="form-input" id="chats-search-contacts" placeholder="البحث في جهات الاتصال..." oninput="filterChatContacts()">
                            </div>
                            <div class="chat-list-scroll" id="chats-contacts-list">
                                <!-- Dynamic chats list -->
                            </div>
                        </div>

                        <!-- Chats Main Area -->
                        <div class="chat-content-pane">
                            <div class="chat-pane-header">
                                <div class="chat-pane-title" id="chat-active-title">
                                    <span data-tr="no_active_chat">لا توجد محادثة نشطة</span>
                                </div>
                                <div class="chat-header-actions" id="chat-header-icons" style="display:none;">
                                    <div class="flex-row-gap" style="color:var(--accent-emerald); font-size:0.75rem; font-weight:700;">
                                        <svg viewBox="0 0 24 24" style="width:16px; height:16px; stroke:currentColor; fill:none; stroke-width:2.5;"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                                        <span data-tr="e2e_secured">مؤمنة E2E</span>
                                    </div>
                                </div>
                            </div>

                            <!-- Messages Area -->
                            <div class="chat-messages-scroll" id="chat-messages-box">
                                <div style="text-align:center; color:var(--text-muted); margin-top:6rem;" data-tr="chat_select_hint">
                                    اختر أحد الوكلاء أو جهات الاتصال المتاحة لبدء التراسل الخاص المشفر
                                </div>
                            </div>

                            <!-- Shared files display inside chat -->
                            <div class="shared-files-tab" id="chat-shared-files-box"></div>

                            <!-- Input Area -->
                            <div class="chat-input-row" id="chat-input-row-container" style="display:none;">
                                <div id="chat-typing-indicator" style="font-size:0.75rem; color:var(--accent-cyan); height:16px; font-weight:700;"></div>
                                <div class="chat-input-controls">
                                    <input type="text" class="form-input" id="chat-text-input" placeholder="اكتب رسالتك الخاصة والآمنة..." onkeydown="if(event.key==='Enter') executeSendDirectMessage()">
                                    <button class="btn btn-primary" onclick="executeSendDirectMessage()" data-tr="send">إرسال</button>
                                </div>
                                <div class="chat-drag-zone" onclick="triggerDirectFileUpload()" id="chat-drop-file-zone">
                                    <span data-tr="drag_file_hint">اسحب ملفًا هنا أو انقر للمشاركة المشفرة المباشرة</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 📢 TAB: CHANNELS -->
                <div id="channels-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="channels_title">قنوات البث اللامركزية</h1>
                        <p data-tr="channels_subtitle">قنوات اتصال مفتوحة وعامة بالاعتماد على GossipSub للـ PubSub الموزع</p>
                    </div>

                    <div class="chat-main-container">
                        <!-- Channels Sidebar -->
                        <div class="channel-pane-sidebar">
                            <div class="channel-split-tabs">
                                <button class="channel-split-btn active" onclick="switchChannelCategory('public')" data-tr="chan_public">العامة</button>
                                <button class="channel-split-btn" onclick="switchChannelCategory('private')" data-tr="chan_private">الخاصة</button>
                            </div>
                            <div class="chat-sidebar-search">
                                <input type="text" class="form-input" id="channels-join-input" placeholder="انضمام لقناة (مثال: lobby)..." onkeydown="if(event.key==='Enter') executeJoinNewChannel()">
                            </div>
                            <div class="chat-list-scroll" id="channels-menu-list">
                                <!-- Dynamic channels list -->
                            </div>
                        </div>

                        <!-- Channel Main Area -->
                        <div class="chat-content-pane">
                            <div class="chat-pane-header">
                                <div class="chat-pane-title" id="channel-active-title">
                                    <span data-tr="no_active_channel">لا توجد قناة نشطة</span>
                                </div>
                                <div class="chat-header-actions" id="channel-header-actions" style="display:none;">
                                    <button class="btn btn-secondary" style="padding:0.4rem 0.8rem; font-size:0.75rem;" onclick="toggleMuteChannel()" data-tr="mute">كتم</button>
                                    <button class="btn btn-secondary" style="padding:0.4rem 0.8rem; font-size:0.75rem;" onclick="togglePinChannel()" data-tr="pin">تثبيت</button>
                                </div>
                            </div>

                            <div class="chat-messages-scroll" id="channel-messages-box">
                                <div style="text-align:center; color:var(--text-muted); margin-top:6rem;" data-tr="channel_select_hint">
                                    اختر قناة من الشريط الجانبي أو انضم إلى قناة جديدة للمشاركة في نقاشات الشبكة العامة
                                </div>
                            </div>

                            <div class="chat-input-row" id="channel-input-row-container" style="display:none;">
                                <div class="chat-input-controls">
                                    <input type="text" class="form-input" id="channel-text-input" placeholder="اكتب رسالة في القناة..." onkeydown="if(event.key==='Enter') executeSendChannelMessage()">
                                    <button class="btn btn-primary" onclick="executeSendChannelMessage()" data-tr="send">إرسال</button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 🌐 TAB: BROWSER -->
                <div id="browser-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="browser_title">متصفح النطاقات اللامركزية</h1>
                        <p data-tr="browser_subtitle">عرض واستعراض مواقع الويب والخدمات المشفرة المخزنة بالكامل عبر شبكة Bitswap</p>
                    </div>

                    <div class="browser-container">
                        <!-- Navigation bar toolbar -->
                        <div class="browser-toolbar">
                            <div class="browser-nav-group">
                                <button class="browser-nav-btn" onclick="executeBrowserNav('back')">
                                    <svg viewBox="0 0 24 24"><polyline points="15 18 9 12 15 6"></polyline></svg>
                                </button>
                                <button class="browser-nav-btn" onclick="executeBrowserNav('forward')">
                                    <svg viewBox="0 0 24 24"><polyline points="9 18 15 12 9 6"></polyline></svg>
                                </button>
                                <button class="browser-nav-btn" onclick="executeBrowserNav('refresh')">
                                    <svg viewBox="0 0 24 24"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"></path></svg>
                                </button>
                                <button class="browser-nav-btn" onclick="executeBrowserNav('home')">
                                    <svg viewBox="0 0 24 24"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path></svg>
                                </button>
                            </div>

                            <div class="browser-address-container">
                                <span style="font-size:0.8rem; font-weight:800; color:var(--accent-cyan); font-family:'Outfit';">ia://</span>
                                <input type="text" class="browser-address-input" id="browser-address-bar" value="welcome.ia" onkeydown="if(event.key==='Enter') executeBrowserLoad()">
                                <div class="browser-badge verified" id="browser-safety-badge">
                                    <svg viewBox="0 0 24 24"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                                    <span data-tr="secure">آمن ومحقق</span>
                                </div>
                            </div>
                            <button class="btn btn-primary" style="padding:0.4rem 1.25rem; font-size:0.8rem; border-radius:30px;" onclick="executeBrowserLoad()" data-tr="go">تصفح</button>
                        </div>

                        <!-- Iframe view port -->
                        <div class="browser-viewport" id="browser-iframe-viewport">
                            <div class="browser-mock-home" id="browser-homepage-mock">
                                <svg viewBox="0 0 24 24" style="width:64px; height:64px; stroke:var(--accent-cyan); fill:none; stroke-width:1.5;"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>
                                <h2 style="font-weight:900;" data-tr="welcome_ia">مرحباً بك في بوابة الويب اللامركزي</h2>
                                <p style="max-width:500px; color:var(--text-muted);" data-tr="welcome_ia_desc">
                                    اكتب اسم النطاق المطلوب في شريط العنوان أعلاه، أو تصفح أحد المواقع المفضلة المخزنة محلياً بالأسفل.
                                </p>
                                
                                <div class="browser-history-list" id="browser-fav-sites-list">
                                    <!-- Dynamic history items -->
                                </div>
                            </div>
                            <iframe id="browser-active-iframe" src="" style="width:100%; height:100%; border:none; display:none; background:#ffffff;"></iframe>
                        </div>
                    </div>
                </div>

                <!-- 🔎 TAB: EXPLORE -->
                <div id="explore-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="explore_title">استكشاف محتويات الشبكة</h1>
                        <p data-tr="explore_subtitle">البحث عن خدمات الوكلاء، القنوات الرائجة وأحدث النطاقات المسجلة</p>
                    </div>

                    <div class="grid-3" style="margin-bottom:1.5rem;">
                        <div class="card" style="margin-bottom:0;">
                            <h3 style="font-size:0.95rem; font-weight:800; margin-bottom:0.75rem;" data-tr="popular_channels">أبرز القنوات العامة</h3>
                            <div style="display:flex; flex-direction:column; gap:0.5rem;" id="explore-trending-channels">
                                <!-- Dynamic popular channels -->
                            </div>
                        </div>

                        <div class="card" style="margin-bottom:0;">
                            <h3 style="font-size:0.95rem; font-weight:800; margin-bottom:0.75rem;" data-tr="new_domains">أحدث النطاقات المسجلة</h3>
                            <div style="display:flex; flex-direction:column; gap:0.5rem;" id="explore-recent-domains">
                                <!-- Dynamic recent domains -->
                            </div>
                        </div>

                        <div class="card" style="margin-bottom:0;">
                            <h3 style="font-size:0.95rem; font-weight:800; margin-bottom:0.75rem;" data-tr="index_services">فهرس الخدمات الفعالة</h3>
                            <div style="display:flex; flex-direction:column; gap:0.5rem;" id="explore-public-services">
                                <!-- Dynamic public services -->
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="explore_agents">الوكلاء المتوفرون بالشبكة</h3>
                        <div class="explore-agents-grid" id="explore-agents-list">
                            <!-- Dynamic explore agents cards -->
                        </div>
                    </div>

                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="network_map">خريطة العقد التفاعلية</h3>
                        <div class="network-map-canvas" id="network-graph-container">
                            <!-- SVG elements represent interactive nodes -->
                            <svg width="100%" height="100%" id="explore-nodes-svg" style="position:absolute; top:0; left:0;"></svg>
                        </div>
                    </div>
                </div>

                <!-- 📁 TAB: FILES -->
                <div id="files-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="files_title">مخزن الملفات اللامركزي</h1>
                        <p data-tr="files_subtitle">إدارة وتوزيع الملفات عبر BlockStore الخاص بالوكيل ومزامنتها على شبكة IPFS</p>
                    </div>

                    <div class="card">
                        <div class="file-drop-area" id="files-upload-dragzone" onclick="triggerLocalFileUpload()">
                            <svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>
                            <div>
                                <h3 style="font-weight:800;" data-tr="upload_area_title">اسحب الملفات هنا أو اضغط للاختيار والرفع</h3>
                                <p style="font-size:0.8rem; color:var(--text-muted);" data-tr="upload_area_subtitle">يتم حساب الـ CID تلقائياً وتوزيع كتل البيانات بالشبكة</p>
                            </div>
                            <div id="files-upload-progress" class="hidden" style="width:100%; max-width:300px;">
                                <div class="resource-bar-outer" style="height:12px;">
                                    <div class="resource-bar-inner" id="files-upload-progress-bar" style="width:0%;"></div>
                                </div>
                                <span style="font-size:0.75rem; color:var(--accent-cyan);" id="files-upload-progress-text">0%</span>
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="local_stored_files">الملفات المخزنة محلياً</h3>
                        <div class="table-container">
                            <table>
                                <thead>
                                    <tr>
                                        <th data-tr="file_name">اسم الملف</th>
                                        <th data-tr="file_size">الحجم</th>
                                        <th data-tr="file_cid">معرف CID</th>
                                        <th data-tr="file_replicas">النسخ الاحتياطية</th>
                                        <th data-tr="actions">الإجراءات</th>
                                    </tr>
                                </thead>
                                <tbody id="files-table-body">
                                    <!-- Dynamic files list -->
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>

                <!-- 👥 TAB: CONTACTS -->
                <div id="contacts-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="contacts_title">دفتر جهات الاتصال اللامركزي</h1>
                        <p data-tr="contacts_subtitle">إضافة وإدارة هويات الوكلاء والمستخدمين المعروفين على الشبكة</p>
                    </div>

                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="add_contact_title">إضافة جهة اتصال جديدة</h3>
                        <div class="grid-2">
                            <div class="form-group">
                                <label data-tr="contact_alias_lbl">اسم الشهرة / الاسم المستعار</label>
                                <input type="text" class="form-input" id="contact-add-alias" placeholder="مثال: Alice">
                            </div>
                            <div class="form-group">
                                <label data-tr="contact_did_lbl">معرف الهوية الكامل (DID / .ia)</label>
                                <input type="text" class="form-input" id="contact-add-did" placeholder="did:ia:... أو domain.ia">
                            </div>
                        </div>
                        <button class="btn btn-primary" onclick="executeAddContact()" data-tr="add_btn">إضافة للدفتر</button>
                    </div>

                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="saved_contacts">جهات الاتصال المسجلة</h3>
                        <div class="explore-agents-grid" id="contacts-grid-list">
                            <!-- Dynamic contacts grid -->
                        </div>
                    </div>
                </div>

                <!-- 🛠️ TAB: DEVTOOLS -->
                <div id="devtools-panel" class="tab-panel">
                    <div class="page-header">
                        <h1 data-tr="devtools_title">أدوات المطور والمراقبة</h1>
                        <p data-tr="devtools_subtitle">إدارة النطاقات، إصدار التفويضات ومراقبة السجلات واستهلاك الموارد</p>
                    </div>

                    <div class="grid-2">
                        <!-- Domain Management -->
                        <div class="card">
                            <h3 style="font-size:1rem; font-weight:800; margin-bottom:1.25rem;" data-tr="manage_domains">إدارة نطاقاتي (.ia)</h3>
                            <div class="form-group">
                                <label data-tr="domain_name">اسم النطاق</label>
                                <input type="text" class="form-input" id="devtools-domain-name" placeholder="مثال: custom.ia">
                            </div>
                            <div class="form-group">
                                <label data-tr="domain_manifest">معرف ManifestCID</label>
                                <input type="text" class="form-input" id="devtools-domain-manifest" placeholder="Qm... / bafy...">
                            </div>
                            <button class="btn btn-primary" onclick="executeUpdateDomain()" data-tr="update_domain_btn">تحديث النطاق</button>
                        </div>

                        <!-- Delegation management -->
                        <div class="card">
                            <h3 style="font-size:1rem; font-weight:800; margin-bottom:1.25rem;" data-tr="manage_delegations">تفويض الهوية والصلاحيات</h3>
                            <div style="display:flex; flex-direction:column; gap:0.75rem;">
                                <div class="flex-between" style="font-size:0.85rem; border-bottom:1px solid var(--panel-border); padding-bottom:0.5rem;">
                                    <span data-tr="delegations_sent">تفويضات صادرة</span>
                                    <span style="font-weight:800;" id="devtools-delegations-sent-count">0</span>
                                </div>
                                <div class="flex-between" style="font-size:0.85rem; border-bottom:1px solid var(--panel-border); padding-bottom:0.5rem;">
                                    <span data-tr="delegations_received">تفويضات واردة</span>
                                    <span style="font-weight:800;" id="devtools-delegations-recv-count">0</span>
                                </div>
                                <button class="btn btn-secondary" onclick="openNewDelegationModal()" data-tr="issue_delegation">إصدار تفويض صلاحيات جديد</button>
                            </div>
                        </div>
                    </div>

                    <!-- Resource Monitoring -->
                    <div class="card">
                        <h3 style="font-size:1rem; font-weight:800; margin-bottom:1rem;" data-tr="resource_monitor">مراقبة موارد النظام</h3>
                        <div class="resource-chart-row">
                            <div class="resource-chart-bar-container">
                                <div class="flex-between" style="font-size:0.8rem;">
                                    <span data-tr="cpu_usage">استهلاك المعالج</span>
                                    <span id="devtools-cpu-text">0%</span>
                                </div>
                                <div class="resource-bar-outer">
                                    <div class="resource-bar-inner" id="devtools-cpu-bar"></div>
                                </div>
                            </div>
                            <div class="resource-chart-bar-container">
                                <div class="flex-between" style="font-size:0.8rem;">
                                    <span data-tr="ram_usage">استهلاك الذاكرة العشوائية</span>
                                    <span id="devtools-ram-text">0%</span>
                                </div>
                                <div class="resource-bar-outer">
                                    <div class="resource-bar-inner" id="devtools-ram-bar"></div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Live API Logs -->
                    <div class="card">
                        <div class="flex-between" style="margin-bottom:1rem;">
                            <h3 style="font-size:1rem; font-weight:800;" data-tr="live_logs">سجلات الـ API والاتصال الحية</h3>
                            <button class="btn btn-secondary" style="padding:0.35rem 0.75rem; font-size:0.75rem;" onclick="clearLiveLogs()" data-tr="clear">تفريغ</button>
                        </div>
                        <div class="dev-terminal-view" id="devtools-terminal-logs">
                            <!-- Dynamic logs -->
                        </div>
                    </div>
                </div>

            </div>
        </div>
    </div>

    <!-- MODAL: Issue New Delegation -->
    <div class="modal-overlay" id="modal-new-delegation">
        <div class="modal-content">
            <div class="modal-header">
                <h3 data-tr="issue_delegation">تفويض صلاحيات جديد</h3>
                <button class="close-modal-btn" onclick="closeNewDelegationModal()">
                    <svg viewBox="0 0 24 24" style="width:20px; height:20px; stroke:currentColor; fill:none; stroke-width:2.5;"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                </button>
            </div>
            <div class="form-group">
                <label data-tr="delegation_to_lbl">الوكيل المفوض له (DID)</label>
                <input type="text" class="form-input" id="delegation-to-did" placeholder="did:ia:...">
            </div>
            <div class="form-group">
                <label data-tr="delegation_cap_lbl">الصلاحيات الممنوحة (Capabilities)</label>
                <input type="text" class="form-input" id="delegation-capabilities" placeholder="مثال: read:files, publish:channels">
            </div>
            <div class="form-group">
                <label data-tr="delegation_exp_lbl">تاريخ انتهاء الصلاحية (ساعة)</label>
                <input type="number" class="form-input" id="delegation-expires" value="24">
            </div>
            <div style="display:flex; justify-content:flex-end; gap:0.5rem;">
                <button class="btn btn-secondary" onclick="closeNewDelegationModal()" data-tr="cancel">إلغاء</button>
                <button class="btn btn-primary" onclick="executeIssueDelegation()" data-tr="confirm">تأكيد وإصدار</button>
            </div>
        </div>
    </div>

    <!-- Hidden file input triggers -->
    <input type="file" id="hidden-file-input" style="display:none;" onchange="handleFileUploadSelected(event)">

    <!-- Toast Container -->
    <div class="toast-container" id="toast-container"></div>

    <script>
        // Dictionary for Multilingual Translations
        const translations = {
            ar: {
                tab_home: "الرئيسية",
                tab_chats: "المحادثات",
                tab_channels: "القنوات",
                tab_browser: "المتصفح اللامركزي",
                tab_explore: "استكشاف",
                tab_files: "الملفات",
                tab_contacts: "جهات الاتصال",
                tab_devtools: "أدوات المطور",
                connected_status: "متصل بالشبكة",
                notifications_title: "الإشعارات",
                notif_clear: "تحديد كمقروء",
                profile_view: "الملف الشخصي",
                settings: "الإعدادات",
                logout: "تسجيل الخروج",
                home_title: "الموجز الرئيسي للشبكة",
                home_subtitle: "متابعة نشاط العقدة، إحصائيات النظام وإدارة الهوية اللامركزية",
                passport_created: "تاريخ الإنشاء",
                passport_reputation: "السمعة والوثوقية",
                passport_domains: "النطاقات النشطة",
                copy_did: "نسخ المعرف",
                peers_count: "الوكلاء المتصلون",
                registered_domains_count: "النطاقات بالشبكة",
                local_storage_size: "سعة التخزين المحلي",
                quick_actions: "إجراءات سريعة",
                act_register: "تسجيل نطاق جديد (.ia)",
                act_upload: "رفع ملف جديد",
                act_channel: "إنشاء قناة GossipSub",
                recent_activities: "آخر النشاطات العامة",
                chats_title: "محادثات الوكلاء المشفرة",
                chats_subtitle: "تواصل خاص وسري 1:1 مؤمن بالكامل عبر خوارزميات التشفير E2E",
                no_active_chat: "لا توجد محادثة نشطة",
                e2e_secured: "مؤمنة E2E",
                chat_select_hint: "اختر أحد الوكلاء أو جهات الاتصال المتاحة لبدء التراسل الخاص المشفر",
                send: "إرسال",
                drag_file_hint: "اسحب ملفًا هنا أو انقر للمشاركة المشفرة المباشرة",
                channels_title: "قنوات البث اللامركزية",
                channels_subtitle: "قنوات اتصال مفتوحة وعامة بالاعتماد على GossipSub للـ PubSub الموزع",
                chan_public: "العامة",
                chan_private: "الخاصة",
                no_active_channel: "لا توجد قناة نشطة",
                mute: "كتم",
                pin: "تثبيت",
                channel_select_hint: "اختر قناة من الشريط الجانبي أو انضم إلى قناة جديدة للمشاركة في نقاشات الشبكة العامة",
                browser_title: "متصفح النطاقات اللامركزية",
                browser_subtitle: "عرض واستعراض مواقع الويب والخدمات المشفرة المخزنة بالكامل عبر شبكة Bitswap",
                secure: "آمن ومحقق",
                go: "تصفح",
                welcome_ia: "مرحباً بك في بوابة الويب اللامركزي",
                welcome_ia_desc: "اكتب اسم النطاق المطلوب في شريط العنوان أعلاه، أو تصفح أحد المواقع المفضلة المخزنة محلياً بالأسفل.",
                explore_title: "استكشاف محتويات الشبكة",
                explore_subtitle: "البحث عن خدمات الوكلاء، القنوات الرائجة وأحدث النطاقات المسجلة",
                popular_channels: "أبرز القنوات العامة",
                new_domains: "أحدث النطاقات المسجلة",
                index_services: "فهرس الخدمات الفعالة",
                explore_agents: "الوكلاء المتوفرون بالشبكة",
                network_map: "خريطة العقد التفاعلية",
                files_title: "مخزن الملفات اللامركزي",
                files_subtitle: "إدارة وتوزيع الملفات عبر BlockStore الخاص بالوكيل ومزامنتها على شبكة IPFS",
                upload_area_title: "اسحب الملفات هنا أو اضغط للاختيار والرفع",
                upload_area_subtitle: "يتم حساب الـ CID تلقائياً وتوزيع كتل البيانات بالشبكة",
                local_stored_files: "الملفات المخزنة محلياً",
                file_name: "اسم الملف",
                file_size: "الحجم",
                file_cid: "معرف CID",
                file_replicas: "النسخ الاحتياطية",
                actions: "الإجراءات",
                contacts_title: "دفتر جهات الاتصال اللامركزي",
                contacts_subtitle: "إضافة وإدارة هويات الوكلاء والمستخدمين المعروفين على الشبكة",
                add_contact_title: "إضافة جهة اتصال جديدة",
                contact_alias_lbl: "اسم الشهرة / الاسم المستعار",
                contact_did_lbl: "معرف الهوية الكامل (DID / .ia)",
                add_btn: "إضافة للدفتر",
                saved_contacts: "جهات الاتصال المسجلة",
                devtools_title: "أدوات المطور والمراقبة",
                devtools_subtitle: "إدارة النطاقات، إصدار التفويضات ومراقبة السجلات واستهلاك الموارد",
                manage_domains: "إدارة نطاقاتي (.ia)",
                domain_name: "اسم النطاق",
                domain_manifest: "معرف ManifestCID",
                update_domain_btn: "تحديث النطاق",
                manage_delegations: "تفويض الهوية والصلاحيات",
                delegations_sent: "تفويضات صادرة",
                delegations_received: "تفويضات واردة",
                issue_delegation: "إصدار تفويض صلاحيات جديد",
                resource_monitor: "مراقبة موارد النظام",
                cpu_usage: "استهلاك المعالج",
                ram_usage: "استهلاك الذاكرة العشوائية",
                live_logs: "سجلات الـ API والاتصال الحية",
                clear: "تفريغ",
                delegation_to_lbl: "الوكيل المفوض له (DID)",
                delegation_cap_lbl: "الصلاحيات الممنوحة (Capabilities)",
                delegation_exp_lbl: "تاريخ انتهاء الصلاحية (ساعة)",
                cancel: "إلغاء",
                confirm: "تأكيد وإصدار"
            },
            en: {
                tab_home: "Home",
                tab_chats: "Chats",
                tab_channels: "Channels",
                tab_browser: "Decentralized Browser",
                tab_explore: "Explore",
                tab_files: "Files",
                tab_contacts: "Contacts",
                tab_devtools: "Dev Tools",
                connected_status: "Connected to Network",
                notifications_title: "Notifications",
                notif_clear: "Mark read",
                profile_view: "My Profile",
                settings: "Settings",
                logout: "Logout",
                home_title: "Network Summary Feed",
                home_subtitle: "Monitor agent status, system resources, and manage decentralized identity",
                passport_created: "Registration Date",
                passport_reputation: "Reputation & Trust",
                passport_domains: "Active Domains",
                copy_did: "Copy DID",
                peers_count: "Connected Peers",
                registered_domains_count: "Network Domains",
                local_storage_size: "Local Storage Size",
                quick_actions: "Quick Actions",
                act_register: "Register New Domain (.ia)",
                act_upload: "Upload New File",
                act_channel: "Create GossipSub Channel",
                recent_activities: "Recent General Activities",
                chats_title: "Encrypted Agent Chats",
                chats_subtitle: "Private, E2E-secured 1:1 messaging utilizing cryptographic identity key exchange",
                no_active_chat: "No Active Chat Selected",
                e2e_secured: "E2E Secured",
                chat_select_hint: "Choose a contact from the sidebar list to start a secure encrypted session",
                send: "Send",
                drag_file_hint: "Drag and drop files here or click to securely share via blockstore",
                channels_title: "Decentralized Channels",
                channels_subtitle: "Public and group communication channels relying on GossipSub pubsub network",
                chan_public: "Public",
                chan_private: "Private",
                no_active_channel: "No Active Channel",
                mute: "Mute",
                pin: "Pin",
                channel_select_hint: "Select a channel or join a new room to participate in public discussions",
                browser_title: "Decentralized Browser",
                browser_subtitle: "Browse portals, static content, and services stored entirely on Bitswap network",
                secure: "Verified secure",
                go: "Go",
                welcome_ia: "Welcome to the Decentralized Web Portal",
                welcome_ia_desc: "Type a domain name ending with .ia above, or click one of the favorite local portals below.",
                explore_title: "Explore Network Content",
                explore_subtitle: "Discover decentralized services, trending channels, and recently registered domains",
                popular_channels: "Popular Public Channels",
                new_domains: "Recently Registered Domains",
                index_services: "Active Services Catalog",
                explore_agents: "Discovered Active Agents",
                network_map: "Interactive Node Topology",
                files_title: "Decentralized File Manager",
                files_subtitle: "Store, track, and serve static assets via local blockstore replicated on IPFS",
                upload_area_title: "Drag & drop files here or click to select and upload",
                upload_area_subtitle: "CID is computed locally and chunks are advertised on DHT",
                local_stored_files: "Locally Stored Content Blocks",
                file_name: "File Name",
                file_size: "Size",
                file_cid: "Content CID",
                file_replicas: "Active Replicas",
                actions: "Actions",
                contacts_title: "Decentralized Contacts Book",
                contacts_subtitle: "Add, view, and manage known digital identities and agents on the network",
                add_contact_title: "Add New Contact",
                contact_alias_lbl: "Alias / Display Name",
                contact_did_lbl: "Digital Identifier (DID / .ia)",
                add_btn: "Add to Directory",
                saved_contacts: "Stored Contacts Directory",
                devtools_title: "Developer Tools & Metrics",
                devtools_subtitle: "Configure naming registries, delegate authorization, view live logs & resource metrics",
                manage_domains: "Manage Owned Domains (.ia)",
                domain_name: "Domain Name",
                domain_manifest: "Target ManifestCID",
                update_domain_btn: "Update Domain",
                manage_delegations: "Identity Authorization & Delegations",
                delegations_sent: "Issued Delegations",
                delegations_received: "Received Delegations",
                issue_delegation: "Issue New Capabilities Delegation",
                resource_monitor: "Resource Monitor",
                cpu_usage: "CPU Usage",
                ram_usage: "Memory Allocation",
                live_logs: "Live REST API Request Logs",
                clear: "Clear Logs",
                delegation_to_lbl: "Delegated Agent (DID)",
                delegation_cap_lbl: "Granted Capabilities",
                delegation_exp_lbl: "Expiration Time (hours)",
                cancel: "Cancel",
                confirm: "Confirm & Issue"
            }
        };

        // Application State
        let state = {
            lang: "ar",
            theme: "dark",
            token: "",
            identity: null,
            activeChannel: "",
            activeChatContact: "",
            channels: [],
            chatMessages: {},
            channelMessages: {},
            contacts: [
                { nickname: "Alice Keyholder", did: "did:ia:QysLqvbC4kfgbXbHwuvbxX", status: "online", caps: ["read:files", "chat:1to1"] },
                { nickname: "Founder Bootstrap", did: "did:ia:BootstrapNodeMasterKey2026", status: "online", caps: ["resolve:domain", "register:domain"] },
                { nickname: "Bob Assistant", did: "did:ia:Nd1QCh18i7FZwy9TuiUGpp", status: "offline", caps: ["translate", "chat:1to1"] }
            ],
            files: [
                { name: "index.html", size: "12.4 KB", cid: "bafybeicw22c55e2...", replicas: 5 },
                { name: "logo.png", size: "26.0 KB", cid: "bafybeidwxx99sa...", replicas: 3 }
            ],
            notifications: [
                { id: 1, text: "طلب تفويض وارد من Bob Assistant للحصول على صلاحية قراءة الملفات", time: "قبل دقيقتين", read: false },
                { id: 2, text: "تم تسجيل النطاق alice.ia بنجاح تحت هويتك اللامركزية", time: "قبل ساعة", read: false }
            ],
            recentActivities: [
                { desc: "تم الانضمام لقناة GossipSub العامة #lobby", time: "19:42" },
                { desc: "تم إرسال معاملة التزام (Commit) لتسجيل النطاق node1.ia", time: "18:15" },
                { desc: "تم الاتصال بالوكيل did:ia:QysLqvbC4kfgbXbHwuvbxX بنجاح", time: "17:30" }
            ],
            browserHistory: [
                { name: "welcome.ia", desc: "Musketeers Welcome Page" },
                { name: "search.ia", desc: "Decentralized Search Portal" },
                { name: "chat-gate.ia", desc: "Encrypted Web Chat Portal" }
            ],
            devtoolsLogs: [],
            delegations: { sent: [], recv: [] },
            channelPollInterval: null,
            chatPollInterval: null,
            resourceInterval: null
        };

        // Initialize App on DOM Loaded
        window.addEventListener('DOMContentLoaded', () => {
            // Load saved settings
            state.theme = localStorage.getItem('nr_theme') || 'dark';
            state.lang = localStorage.getItem('nr_lang') || 'ar';
            
            // Set styles and language
            document.documentElement.setAttribute('data-theme', state.theme);
            document.getElementById('theme-toggle-btn').innerText = state.theme === 'dark' ? 'Light' : 'Dark';
            
            updateDocumentDirection();
            translateUI();

            // Handle API token
            const urlParams = new URLSearchParams(window.location.search);
            const token = urlParams.get('token');
            if (token) {
                localStorage.setItem('nr_token', token);
                state.token = token;
                window.history.replaceState({}, document.title, window.location.pathname);
            } else {
                state.token = localStorage.getItem('nr_token') || "";
            }

            initializeWorkspaceData();
            startPollingLoops();
        });

        // Toggle Theme (Dark / Light)
        function toggleTheme() {
            state.theme = state.theme === 'dark' ? 'light' : 'dark';
            localStorage.setItem('nr_theme', state.theme);
            document.documentElement.setAttribute('data-theme', state.theme);
            document.getElementById('theme-toggle-btn').innerText = state.theme === 'dark' ? 'Light' : 'Dark';
            logAPIRequest("THEME_CHANGED", "Switched layout appearance theme to " + state.theme);
        }

        // Toggle Language (AR / EN)
        function toggleLanguage() {
            state.lang = state.lang === 'ar' ? 'en' : 'ar';
            localStorage.setItem('nr_lang', state.lang);
            document.getElementById('lang-toggle-btn').innerText = state.lang === 'ar' ? 'English' : 'العربية';
            
            updateDocumentDirection();
            translateUI();
            logAPIRequest("LANG_CHANGED", "Switched locale direction configuration to " + state.lang);
        }

        function updateDocumentDirection() {
            if (state.lang === 'ar') {
                document.documentElement.setAttribute('dir', 'rtl');
                document.documentElement.setAttribute('lang', 'ar');
            } else {
                document.documentElement.setAttribute('dir', 'ltr');
                document.documentElement.setAttribute('lang', 'en');
            }
        }

        // Translate elements based on data-tr attributes
        function translateUI() {
            document.querySelectorAll('[data-tr]').forEach(el => {
                const trKey = el.getAttribute('data-tr');
                const translationText = translations[state.lang][trKey];
                if (translationText) {
                    // Check if it's an input placeholder or standard innerText
                    if (el.tagName === 'INPUT') {
                        el.placeholder = translationText;
                    } else {
                        // Keep SVG if exists, replace text node
                        const svg = el.querySelector('svg');
                        if (svg) {
                            el.innerHTML = "";
                            el.appendChild(svg);
                            const textNode = document.createTextNode(" " + translationText);
                            el.appendChild(textNode);
                        } else {
                            el.innerText = translationText;
                        }
                    }
                }
            });
        }

        // Switch Workspace Tabs
        function switchTab(tabId, btnElement = null) {
            document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
            document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
            
            const targetPanel = document.getElementById(tabId + '-panel');
            if (targetPanel) targetPanel.classList.add('active');

            if (btnElement) {
                btnElement.classList.add('active');
            } else {
                // Find nav item button automatically
                document.querySelectorAll('.nav-item').forEach(btn => {
                    const clickAttr = btn.getAttribute('onclick');
                    if (clickAttr && clickAttr.includes(tabId)) {
                        btn.classList.add('active');
                    }
                });
            }
            
            logAPIRequest("UI_TAB_SWITCH", "Opened workspace view tab: " + tabId);
        }

        // API Request Helper
        async function fetchAPI(endpoint, method = "GET", body = null) {
            if (!state.token) {
                showToast("رمز الوصول غير معرف. الرجاء مصادقة الوكيل.", "error");
                return null;
            }

            const url = window.location.origin + endpoint;
            const headers = {
                'Authorization': 'Bearer ' + state.token,
                'Content-Type': 'application/json'
            };

            const init = { method, headers };
            if (body) {
                init.body = typeof body === 'string' ? body : JSON.stringify(body);
            }

            try {
                const startTime = Date.now();
                const res = await fetch(url, init);
                const duration = Date.now() - startTime;
                
                logAPIRequest(method + " " + endpoint, "Duration: " + duration + "ms | Status: " + res.status);

                if (res.status === 401) {
                    showToast("غير مصرح - رمز REST API غير صالح", "error");
                    return null;
                }
                
                if (!res.ok) {
                    const errorText = await res.text();
                    throw new Error(errorText || res.statusText);
                }

                const contentType = res.headers.get("content-type");
                if (contentType && contentType.includes("application/json")) {
                    return await res.json();
                }
                return await res.text();
            } catch (err) {
                logAPIRequest(method + " " + endpoint + " ERROR", err.message);
                return null;
            }
        }

        // Live Log terminal renderer
        function logAPIRequest(action, detail) {
            const timestamp = new Date().toISOString().substring(11, 19);
            const logMsg = "[" + timestamp + "] " + action + " -> " + detail;
            state.devtoolsLogs.push(logMsg);
            if (state.devtoolsLogs.length > 50) state.devtoolsLogs.shift();
            
            const logBox = document.getElementById('devtools-terminal-logs');
            if (logBox) {
                const div = document.createElement('div');
                div.innerText = logMsg;
                if (action.includes("ERROR")) {
                    div.style.color = "var(--accent-rose)";
                } else if (action.includes("POST") || action.includes("PUT")) {
                    div.style.color = "var(--accent-purple)";
                } else {
                    div.style.color = "#22c55e";
                }
                logBox.appendChild(div);
                logBox.scrollTop = logBox.scrollHeight;
            }
        }

        function clearLiveLogs() {
            state.devtoolsLogs = [];
            const logBox = document.getElementById('devtools-terminal-logs');
            if (logBox) logBox.innerHTML = "";
        }

        // Initialize node identity
        async function initializeWorkspaceData() {
            const identity = await fetchAPI("/api/identity");
            if (identity) {
                state.identity = identity;
                
                // Set identity UI
                document.getElementById('sb-profile-name').innerText = identity.did.substring(0, 16) + "...";
                document.getElementById('sb-avatar-letter').innerText = identity.did.substring(8, 9).toUpperCase();
                document.getElementById('topbar-avatar-letter').innerText = identity.did.substring(8, 9).toUpperCase();
                document.getElementById('home-passport-avatar').innerText = identity.did.substring(8, 9).toUpperCase();
                document.getElementById('home-passport-did').innerText = identity.did;
                
                const created = new Date(identity.created_at * 1000).toLocaleDateString(state.lang === 'ar' ? 'ar-EG' : 'en-US');
                document.getElementById('home-passport-created').innerText = created;
                
                // Auto join Lobby
                await fetchAPI("/api/channels/join", "POST", { channel_id: "lobby" });
                
                // Load views
                renderHomeActivities();
                renderChatsSidebar();
                renderChannelsMenu();
                renderExploreView();
                renderFilesTable();
                renderContactsGrid();
                renderNotifications();
            } else {
                showToast("فشل في استرداد الهوية للوكيل. يرجى إعادة تحميل الصفحة برمز توكين صالح.", "error");
            }
        }

        // Start Background Polling Loops
        function startPollingLoops() {
            // Resource usage simulator
            state.resourceInterval = setInterval(() => {
                const cpuVal = Math.floor(Math.random() * 25) + 5;
                const ramVal = Math.floor(Math.random() * 15) + 40;
                
                const cpuBar = document.getElementById('devtools-cpu-bar');
                const ramBar = document.getElementById('devtools-ram-bar');
                const cpuText = document.getElementById('devtools-cpu-text');
                const ramText = document.getElementById('devtools-ram-text');

                if (cpuBar) cpuBar.style.width = cpuVal + "%";
                if (ramBar) ramBar.style.width = ramVal + "%";
                if (cpuText) cpuText.innerText = cpuVal + "%";
                if (ramText) ramText.innerText = ramVal + "%";

                // Update peer stat dynamically
                updatePeersCount();
            }, 3000);

            // Channel message refresh
            state.channelPollInterval = setInterval(async () => {
                if (state.activeChannel) {
                    await refreshChannelMessages(state.activeChannel);
                }
            }, 2000);

            // Chat message simulator
            state.chatPollInterval = setInterval(() => {
                if (state.activeChatContact) {
                    simulateIncomingDirectChat();
                }
            }, 8000);
        }

        async function updatePeersCount() {
            // Check connected peers via REST if supported, otherwise fetch stats
            const res = await fetchAPI("/api/health");
            if (res) {
                // Mock peers increment/decrement to show network activity
                const peersCount = Math.floor(Math.random() * 3) + 7;
                document.getElementById('home-stat-peers').innerText = peersCount;
            }
        }

        // Render functions
        function renderHomeActivities() {
            const box = document.getElementById('home-activities-list');
            if (!box) return;
            box.innerHTML = "";
            state.recentActivities.forEach(act => {
                const div = document.createElement('div');
                div.className = "recent-item";
                div.innerHTML = '<span>' + act.desc + '</span><span class="passport-key">' + act.time + '</span>';
                box.appendChild(div);
            });
        }

        // --- Notifications Center ---
        function toggleNotifDropdown() {
            const menu = document.getElementById('notif-dropdown-menu');
            const isVisible = menu.style.display === 'block';
            
            // Close other menus
            document.getElementById('profile-dropdown-menu').style.display = 'none';
            document.getElementById('search-autocomplete-box').style.display = 'none';

            menu.style.display = isVisible ? 'none' : 'block';
        }

        function renderNotifications() {
            const list = document.getElementById('notif-items-list');
            const badge = document.getElementById('notif-count-badge');
            if (!list) return;
            list.innerHTML = "";
            
            const unread = state.notifications.filter(n => !n.read).length;
            badge.innerText = unread;
            badge.style.display = unread > 0 ? 'flex' : 'none';

            if (state.notifications.length === 0) {
                list.innerHTML = '<div style="padding:1.5rem; text-align:center; color:var(--text-muted); font-size:0.8rem;">لا توجد إشعارات حالياً</div>';
                return;
            }

            state.notifications.forEach(n => {
                const div = document.createElement('div');
                div.className = "notif-item";
                if (!n.read) div.style.background = "rgba(6, 182, 212, 0.04)";
                div.onclick = () => markNotifAsRead(n.id);
                div.innerHTML = '<span class="notif-text">' + n.text + '</span><span class="notif-time">' + n.time + '</span>';
                list.appendChild(div);
            });
        }

        function markNotifAsRead(id) {
            const n = state.notifications.find(not => not.id === id);
            if (n) {
                n.read = true;
                renderNotifications();
                showToast("تم تحديد الإشعار كمقروء", "success");
            }
        }

        function clearNotifications() {
            state.notifications.forEach(n => n.read = true);
            renderNotifications();
            showToast("تم قراءة جميع الإشعارات", "success");
            document.getElementById('notif-dropdown-menu').style.display = 'none';
        }

        function toggleProfileDropdown() {
            const menu = document.getElementById('profile-dropdown-menu');
            const isVisible = menu.style.display === 'block';
            document.getElementById('notif-dropdown-menu').style.display = 'none';
            menu.style.display = isVisible ? 'none' : 'block';
        }

        function logoutSession() {
            localStorage.removeItem('nr_token');
            showToast("تم تسجيل الخروج بنجاح", "success");
            setTimeout(() => window.location.reload(), 1000);
        }

        // --- 🔍 Universal Search and Auto-Complete ---
        function handleUniversalSearchInput() {
            const input = document.getElementById('topbar-search-input');
            const query = input.value.trim().toLowerCase();
            const box = document.getElementById('search-autocomplete-box');
            
            if (!query) {
                box.style.display = 'none';
                return;
            }

            // Search entities in local dataset + domains
            const suggestions = [];
            
            // Search contacts
            state.contacts.forEach(c => {
                if (c.nickname.toLowerCase().includes(query) || c.did.toLowerCase().includes(query)) {
                    suggestions.push({ title: c.nickname, type: "Agent / Wekeel", action: () => openDirectChat(c.did) });
                }
            });

            // Search history portals
            state.browserHistory.forEach(h => {
                if (h.name.toLowerCase().includes(query) || h.desc.toLowerCase().includes(query)) {
                    suggestions.push({ title: h.name, type: "Domain (.ia)", action: () => openDomainInBrowser(h.name) });
                }
            });

            // Add standard lobby
            if ("lobby".includes(query)) {
                suggestions.push({ title: "# lobby", type: "Channel", action: () => openChannelRoom("lobby") });
            }

            if (suggestions.length === 0) {
                box.innerHTML = '<div style="padding:1rem; text-align:center; color:var(--text-muted); font-size:0.8rem;">لا توجد اقتراحات مطابقة. اضغط Enter للبحث الشامل...</div>';
            } else {
                box.innerHTML = "";
                suggestions.slice(0, 5).forEach(s => {
                    const item = document.createElement('div');
                    item.className = "autocomplete-item";
                    item.onclick = () => {
                        s.action();
                        box.style.display = 'none';
                        input.value = "";
                    };
                    item.innerHTML = '<span class="autocomplete-title">' + s.title + '</span><span class="autocomplete-type">' + s.type + '</span>';
                    box.appendChild(item);
                });
            }

            box.style.display = 'block';
        }

        async function executeUniversalSearch() {
            const input = document.getElementById('topbar-search-input');
            const query = input.value.trim();
            if (!query) return;

            document.getElementById('search-autocomplete-box').style.display = 'none';
            showToast("جاري الاستعلام الشامل في فهارس DHT...", "warning");

            // Redirect to resolve or explore tab
            switchTab('explore');
            const inputExplore = document.getElementById('devtools-domain-name');
            if (inputExplore) inputExplore.value = query;

            // Resolve domain
            const res = await fetchAPI("/api/resolve?name=" + encodeURIComponent(query));
            if (res && res.owner) {
                showToast("تم العثور على نطاق مطابق: " + query, "success");
                openDomainInBrowser(query);
            } else {
                showToast("البحث لم يرجع نتائج مباشرة. تم نشر إعلان الاستعلام.", "warning");
            }
            input.value = "";
        }

        // Close dropdowns when clicking outside
        window.addEventListener('click', (e) => {
            if (!e.target.closest('.notif-bell-container')) {
                document.getElementById('notif-dropdown-menu').style.display = 'none';
            }
            if (!e.target.closest('.topbar-profile-container')) {
                document.getElementById('profile-dropdown-menu').style.display = 'none';
            }
            if (!e.target.closest('.universal-search-container')) {
                document.getElementById('search-autocomplete-box').style.display = 'none';
            }
        });

        // --- 💬 CHAT DIRECT MESSAGES SECTION ---
        function renderChatsSidebar() {
            const list = document.getElementById('chats-contacts-list');
            if (!list) return;
            list.innerHTML = "";
            state.contacts.forEach(c => {
                const isActive = state.activeChatContact === c.did;
                const div = document.createElement('div');
                div.className = "chat-item-card" + (isActive ? " active" : "");
                div.onclick = () => openDirectChat(c.did);

                const avatar = document.createElement('div');
                avatar.className = "avatar-circle";
                avatar.style.width = "34px";
                avatar.style.height = "34px";
                avatar.innerText = c.nickname.substring(0, 1).toUpperCase();

                const isOnline = c.status === 'online';
                const statusDot = '<span style="width:6px; height:6px; border-radius:50%; background:' + (isOnline ? 'var(--accent-emerald)' : 'var(--text-muted)') + '; display:inline-block;"></span>';

                div.innerHTML = '';
                div.appendChild(avatar);
                
                const info = document.createElement('div');
                info.className = "chat-item-info";
                info.innerHTML = '<span class="chat-item-name">' + c.nickname + '</span>' +
                                 '<span class="chat-item-preview">' + statusDot + ' ' + (isOnline ? 'Active' : 'Offline') + '</span>';
                div.appendChild(info);
                list.appendChild(div);
            });
        }

        function openDirectChat(did) {
            state.activeChatContact = did;
            switchTab('chats');
            renderChatsSidebar();

            const contact = state.contacts.find(c => c.did === did);
            document.getElementById('chat-active-title').innerText = contact ? contact.nickname : did.substring(0, 18) + "...";
            document.getElementById('chat-header-icons').style.display = "flex";
            document.getElementById('chat-input-row-container').style.display = "flex";

            renderDirectMessagesList();
        }

        function renderDirectMessagesList() {
            const box = document.getElementById('chat-messages-box');
            if (!box) return;
            box.innerHTML = "";

            const msgs = state.chatMessages[state.activeChatContact] || [];
            if (msgs.length === 0) {
                box.innerHTML = '<div style="text-align:center; color:var(--text-muted); margin-top:6rem;">لا توجد رسائل سابقة. أرسل رسالة لبدء المحادثة الآمنة.</div>';
                return;
            }

            msgs.forEach(m => {
                const bubble = document.createElement('div');
                const isSelf = m.sender === 'self';
                bubble.className = "message-bubble " + (isSelf ? "outgoing" : "incoming");
                
                // Action options
                const actions = document.createElement('div');
                actions.className = "msg-actions-hover";
                actions.innerHTML = '<button class="msg-action-btn" onclick="replyMessage(\'' + m.id + '\')"><svg viewBox="0 0 24 24"><polyline points="9 17 4 12 9 7"></polyline><path d="M20 18v-2a4 4 0 0 0-4-4H4"></path></svg></button>' +
                                    '<button class="msg-action-btn" onclick="deleteMessage(\'' + m.id + '\')"><svg viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg></button>';
                bubble.appendChild(actions);

                const textSpan = document.createElement('span');
                textSpan.innerText = m.content;
                bubble.appendChild(textSpan);

                const timeSpan = document.createElement('span');
                timeSpan.className = "passport-key";
                timeSpan.style.fontSize = "0.65rem";
                timeSpan.style.textAlign = "end";
                timeSpan.innerText = m.time;
                bubble.appendChild(timeSpan);

                box.appendChild(bubble);
            });
            box.scrollTop = box.scrollHeight;
        }

        function executeSendDirectMessage() {
            const input = document.getElementById('chat-text-input');
            const text = input.value.trim();
            if (!text || !state.activeChatContact) return;

            if (!state.chatMessages[state.activeChatContact]) {
                state.chatMessages[state.activeChatContact] = [];
            }

            const time = new Date().toLocaleTimeString(state.lang === 'ar' ? 'ar-EG' : 'en-US', { hour: '2-digit', minute: '2-digit' });
            state.chatMessages[state.activeChatContact].push({
                id: Math.random().toString(),
                sender: "self",
                content: text,
                time
            });

            input.value = "";
            renderDirectMessagesList();
            
            // Trigger auto typing simulation
            showTypingIndicator();
        }

        function showTypingIndicator() {
            const ind = document.getElementById('chat-typing-indicator');
            if (!ind) return;
            ind.innerText = "الطرف الآخر يكتب الآن...";
            setTimeout(() => {
                ind.innerText = "";
            }, 3000);
        }

        function simulateIncomingDirectChat() {
            // Simulated reply logic to make the app alive
            const contact = state.contacts.find(c => c.did === state.activeChatContact);
            if (!contact || contact.status !== 'online') return;

            if (!state.chatMessages[state.activeChatContact]) return;
            
            // Check if last message was from self
            const msgs = state.chatMessages[state.activeChatContact];
            if (msgs.length === 0 || msgs[msgs.length - 1].sender !== 'self') return;

            const replies = [
                "أهلاً بك، تم استلام رسالتك المشفرة بنجاح.",
                "أنا أعمل حالياً على معالجة الطلبات في عقدتي.",
                "وصلني الملف، سأتحقق من صحة التوقيع الرقمي.",
                "النظام مستقر تماماً ومعدل الاتصال P2P ممتاز."
            ];
            const content = replies[Math.floor(Math.random() * replies.length)];
            const time = new Date().toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' });
            
            msgs.push({
                id: Math.random().toString(),
                sender: contact.did,
                content,
                time
            });
            
            renderDirectMessagesList();
            
            // Push Notification
            state.notifications.unshift({
                id: Date.now(),
                text: "رسالة مباشرة جديدة من " + contact.nickname,
                time: "الآن",
                read: false
            });
            renderNotifications();
        }

        function replyMessage(id) {
            showToast("تم الرد على الرسالة بنجاح", "success");
        }

        function deleteMessage(id) {
            const msgs = state.chatMessages[state.activeChatContact] || [];
            state.chatMessages[state.activeChatContact] = msgs.filter(m => m.id !== id);
            renderDirectMessagesList();
            showToast("تم حذف الرسالة محلياً", "warning");
        }

        function triggerDirectFileUpload() {
            document.getElementById('hidden-file-input').click();
        }

        function handleFileUploadSelected(e) {
            const file = e.target.files[0];
            if (!file) return;

            showToast("جاري تشفير ونقل الملف اللامركزي: " + file.name, "warning");
            
            // Simulate direct progress bar
            let progress = 0;
            const interval = setInterval(() => {
                progress += 20;
                if (progress >= 100) {
                    clearInterval(interval);
                    showToast("اكتمل نقل الملف الآمن بنجاح", "success");
                    
                    // Add file to chat
                    if (state.activeChatContact) {
                        if (!state.chatMessages[state.activeChatContact]) {
                            state.chatMessages[state.activeChatContact] = [];
                        }
                        state.chatMessages[state.activeChatContact].push({
                            id: Math.random().toString(),
                            sender: "self",
                            content: "📎 ملف مشترك: " + file.name + " (" + (file.size / 1024).toFixed(1) + " KB)",
                            time: new Date().toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' })
                        });
                        renderDirectMessagesList();
                    }
                }
            }, 500);
        }

        // --- 📢 GOSSIPSUB CHANNELS SECTION ---
        async function renderChannelsMenu() {
            const list = document.getElementById('channels-menu-list');
            if (!list) return;
            list.innerHTML = "";

            const joined = await fetchAPI("/api/channels/list");
            if (joined) {
                state.channels = joined;
                if (joined.length === 0) {
                    list.innerHTML = '<div style="padding:1.5rem; text-align:center; color:var(--text-muted); font-size:0.8rem;">لم تنضم لأي قناة بعد</div>';
                    return;
                }

                joined.forEach(ch => {
                    const isActive = state.activeChannel === ch;
                    const div = document.createElement('div');
                    div.className = "chat-item-card" + (isActive ? " active" : "");
                    div.onclick = () => openChannelRoom(ch);

                    const avatar = document.createElement('div');
                    avatar.className = "avatar-circle";
                    avatar.style.width = "34px";
                    avatar.style.height = "34px";
                    avatar.style.background = "linear-gradient(135deg, var(--accent-purple), var(--accent-cyan))";
                    avatar.innerText = ch.substring(0, 1).toUpperCase();

                    const info = document.createElement('div');
                    info.className = "chat-item-info";
                    info.innerHTML = '<span class="chat-item-name"># ' + ch + '</span>' +
                                     '<span class="chat-item-preview">GossipSub Channel</span>';
                    
                    div.appendChild(avatar);
                    div.appendChild(info);
                    list.appendChild(div);
                });
            }
        }

        async function executeJoinNewChannel() {
            const input = document.getElementById('channels-join-input');
            const channelId = input.value.trim().toLowerCase();
            if (!channelId) return;

            showToast("جاري الانضمام إلى قناة GossipSub...", "warning");
            const res = await fetchAPI("/api/channels/join", "POST", { channel_id: channelId });
            if (res) {
                showToast("تم الانضمام لقناة #" + channelId, "success");
                input.value = "";
                await renderChannelsMenu();
                openChannelRoom(channelId);
            }
        }

        function openChannelRoom(ch) {
            state.activeChannel = ch;
            switchTab('channels');
            renderChannelsMenu();

            document.getElementById('channel-active-title').innerText = "# " + ch;
            document.getElementById('channel-header-actions').style.display = "flex";
            document.getElementById('channel-input-row-container').style.display = "flex";

            refreshChannelMessages(ch);
        }

        async function refreshChannelMessages(ch) {
            const msgs = await fetchAPI("/api/channels/messages?channel_id=" + encodeURIComponent(ch));
            if (msgs) {
                state.channelMessages[ch] = msgs;
                renderChannelMessagesList(ch);
            }
        }

        function renderChannelMessagesList(ch) {
            const box = document.getElementById('channel-messages-box');
            if (!box || state.activeChannel !== ch) return;

            const msgs = state.channelMessages[ch] || [];
            const isAtBottom = box.scrollHeight - box.clientHeight <= box.scrollTop + 40;
            
            box.innerHTML = "";
            if (msgs.length === 0) {
                box.innerHTML = '<div style="text-align:center; color:var(--text-muted); margin-top:6rem;">القناة فارغة. كن أول من يرسل رسالة!</div>';
                return;
            }

            const myDID = state.identity ? state.identity.did : "";

            msgs.forEach(m => {
                const bubble = document.createElement('div');
                const senderVal = m.from || m.sender || "";
                const isSelf = senderVal === myDID;
                bubble.className = "message-bubble " + (isSelf ? "outgoing" : "incoming");

                const senderLbl = document.createElement('span');
                senderLbl.className = "msg-sender-lbl";
                senderLbl.innerText = isSelf ? "أنا (هويتي)" : senderVal.substring(0, 15) + "...";
                bubble.appendChild(senderLbl);

                const textSpan = document.createElement('span');
                // Highlight @mentions if match myDID abbreviation
                let contentText = m.content;
                if (contentText.includes("@")) {
                    textSpan.style.color = "var(--accent-amber)";
                }
                textSpan.innerText = contentText;
                bubble.appendChild(textSpan);

                const timeSpan = document.createElement('span');
                timeSpan.className = "passport-key";
                timeSpan.style.fontSize = "0.65rem";
                timeSpan.style.textAlign = "end";
                timeSpan.innerText = new Date(m.timestamp * 1000).toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' });
                bubble.appendChild(timeSpan);

                box.appendChild(bubble);
            });

            if (isAtBottom) {
                box.scrollTop = box.scrollHeight;
            }
        }

        async function executeSendChannelMessage() {
            const input = document.getElementById('channel-text-input');
            const text = input.value.trim();
            if (!text || !state.activeChannel) return;

            const res = await fetchAPI("/api/channels/publish", "POST", {
                channel_id: state.activeChannel,
                content: text
            });

            if (res) {
                input.value = "";
                await refreshChannelMessages(state.activeChannel);
                
                // Add to activities list
                state.recentActivities.unshift({
                    desc: "أرسلت رسالة في القناة #" + state.activeChannel,
                    time: new Date().toTimeString().substring(0, 5)
                });
                renderHomeActivities();
            }
        }

        function toggleMuteChannel() {
            showToast("تم كتم إشعارات القناة مؤقتاً", "warning");
        }

        function togglePinChannel() {
            showToast("تم تثبيت القناة بأعلى القائمة الجانبية", "success");
        }

        // --- 🌐 DECENTRALIZED WEB BROWSER SECTION ---
        function executeBrowserLoad() {
            const input = document.getElementById('browser-address-bar');
            let url = input.value.trim();
            if (!url) return;

            if (!url.endsWith(".ia")) {
                url += ".ia";
                input.value = url;
            }

            showToast("جاري الاستعلام وتوجيه البوابة اللامركزية...", "warning");
            logAPIRequest("RESOLVE_DOMAIN", "Checking Bitswap manifest path for domain target: " + url);

            // Hide mock homepage and load iframe
            document.getElementById('browser-homepage-mock').style.display = 'none';
            const iframe = document.getElementById('browser-active-iframe');
            iframe.style.display = 'block';

            // Point iframe to local http gateway gateway (default 8090)
            const targetUrl = "http://127.0.0.1:8090/d/" + url + "/";
            iframe.src = targetUrl;
            
            // Set verified safety badge based on mock verification
            const safetyBadge = document.getElementById('browser-safety-badge');
            if (url === 'welcome.ia' || url === 'search.ia') {
                safetyBadge.className = 'browser-badge verified';
                safetyBadge.innerHTML = '<svg viewBox="0 0 24 24"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg> <span>آمن ومحقق</span>';
            } else {
                safetyBadge.className = 'browser-badge unverified';
                safetyBadge.innerHTML = '<svg viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg> <span>غير محقق التوقيع</span>';
            }

            // Save to browser history if not exists
            if (!state.browserHistory.some(h => h.name === url)) {
                state.browserHistory.unshift({ name: url, desc: "Resolved site" });
                renderFavPortals();
            }
        }

        function openDomainInBrowser(domain) {
            document.getElementById('browser-address-bar').value = domain;
            switchTab('browser');
            executeBrowserLoad();
        }

        function renderFavPortals() {
            const box = document.getElementById('browser-fav-sites-list');
            if (!box) return;
            box.innerHTML = "";
            state.browserHistory.forEach(h => {
                const item = document.createElement('div');
                item.className = "history-item";
                item.onclick = () => openDomainInBrowser(h.name);
                item.innerHTML = '<span>' + h.name + '</span><span class="passport-key">' + h.desc + '</span>';
                box.appendChild(item);
            });
        }

        function executeBrowserNav(direction) {
            showToast("تنقل المتصفح: " + direction, "warning");
            if (direction === 'home') {
                document.getElementById('browser-active-iframe').style.display = 'none';
                document.getElementById('browser-homepage-mock').style.display = 'flex';
                document.getElementById('browser-address-bar').value = "";
            } else if (direction === 'refresh') {
                const iframe = document.getElementById('browser-active-iframe');
                iframe.src = iframe.src;
            }
        }

        // --- 🔎 EXPLORE TAB SECTION ---
        function renderExploreView() {
            // Render trending channels
            const chBox = document.getElementById('explore-trending-channels');
            chBox.innerHTML = '<div class="recent-item" onclick="openChannelRoom(\'lobby\')"><span># lobby</span><span class="passport-key">نشط جداً</span></div>' +
                               '<div class="recent-item" onclick="openChannelRoom(\'tech\')"><span># tech</span><span class="passport-key">متاح</span></div>';

            // Render recent domains
            const domBox = document.getElementById('explore-recent-domains');
            domBox.innerHTML = '<div class="recent-item" onclick="openDomainInBrowser(\'alice.ia\')"><span>alice.ia</span><span class="passport-key">بواسطة Alice</span></div>' +
                               '<div class="recent-item" onclick="openDomainInBrowser(\'bob.ia\')"><span>bob.ia</span><span class="passport-key">بواسطة Bob</span></div>';

            // Render services
            const srvBox = document.getElementById('explore-public-services');
            srvBox.innerHTML = '<div class="recent-item"><span>ACP Translation Engine</span><span class="passport-key" style="color:var(--accent-emerald);">Active</span></div>' +
                               '<div class="recent-item"><span> BadgerDB Storage Hub</span><span class="passport-key" style="color:var(--accent-emerald);">Active</span></div>';

            // Render Discovered agents
            const agList = document.getElementById('explore-agents-list');
            if (agList) {
                agList.innerHTML = "";
                state.contacts.forEach(c => {
                    const card = document.createElement('div');
                    card.className = "explore-agent-card";
                    card.innerHTML = '<div style="font-weight:800; font-size:0.9rem; color:var(--accent-cyan);">' + c.nickname + '</div>' +
                                     '<div style="font-size:0.7rem; font-family:monospace; color:var(--text-muted); overflow:hidden; text-overflow:ellipsis;">' + c.did + '</div>' +
                                     '<div style="font-size:0.75rem;">Capabilities: ' + c.caps.join(", ") + '</div>' +
                                     '<button class="btn btn-secondary" style="font-size:0.75rem; padding:0.35rem 0.75rem; margin-top:0.5rem;" onclick="openDirectChat(\'' + c.did + '\')">مراسلة الوكيل</button>';
                    agList.appendChild(card);
                });
            }

            renderFavPortals();
            renderNetworkGraph();
        }

        function renderNetworkGraph() {
            const svg = document.getElementById('explore-nodes-svg');
            if (!svg) return;
            svg.innerHTML = "";

            const nodes = [
                { id: "Seed Node", x: 180, y: 150, color: "var(--accent-purple)", size: 10 },
                { id: "Agent 1", x: 100, y: 80, color: "var(--accent-cyan)", size: 8 },
                { id: "Agent 2", x: 260, y: 220, color: "var(--accent-cyan)", size: 8 }
            ];

            // Render paths
            nodes.forEach((n, idx) => {
                if (idx > 0) {
                    const line = document.createElementNS("http://www.w3.org/2000/svg", "line");
                    line.setAttribute("x1", nodes[0].x);
                    line.setAttribute("y1", nodes[0].y);
                    line.setAttribute("x2", n.x);
                    line.setAttribute("y2", n.y);
                    line.setAttribute("stroke", "var(--panel-border)");
                    line.setAttribute("stroke-width", "2");
                    svg.appendChild(line);
                }
            });

            // Render circles
            nodes.forEach(n => {
                const g = document.createElementNS("http://www.w3.org/2000/svg", "g");
                
                const circle = document.createElementNS("http://www.w3.org/2000/svg", "circle");
                circle.setAttribute("cx", n.x);
                circle.setAttribute("cy", n.y);
                circle.setAttribute("r", n.size);
                circle.setAttribute("fill", n.color);
                circle.setAttribute("class", "glowing-circle");

                const text = document.createElementNS("http://www.w3.org/2000/svg", "text");
                text.setAttribute("x", n.x);
                text.setAttribute("y", n.y - 12);
                text.setAttribute("fill", "var(--text-main)");
                text.setAttribute("font-size", "10px");
                text.setAttribute("text-anchor", "middle");
                text.textContent = n.id;

                g.appendChild(circle);
                g.appendChild(text);
                svg.appendChild(g);
            });
        }

        // --- 📁 FILE MANAGER BLOCKSTORE SECTION ---
        function renderFilesTable() {
            const tbody = document.getElementById('files-table-body');
            if (!tbody) return;
            tbody.innerHTML = "";

            state.files.forEach(f => {
                const tr = document.createElement('tr');
                tr.innerHTML = '<td>' + f.name + '</td>' +
                               '<td>' + f.size + '</td>' +
                               '<td style="font-family:monospace; color:var(--accent-cyan);">' + f.cid + '</td>' +
                               '<td>' + f.replicas + ' Nodes</td>' +
                               '<td>' +
                                 '<div style="display:flex; gap:0.5rem;">' +
                                   '<button class="btn btn-secondary" style="font-size:0.7rem; padding:0.25rem 0.5rem;" onclick="copyToClipboardText(\'ia://' + f.cid + '\', \'تم نسخ رابط ia://\')">مشاركة</button>' +
                                   '<button class="btn btn-danger" style="font-size:0.7rem; padding:0.25rem 0.5rem;" onclick="deleteLocalFile(\'' + f.cid + '\')">حذف</button>' +
                                 '</div>' +
                               '</td>';
                tbody.appendChild(tr);
            });
        }

        function triggerLocalFileUpload() {
            document.getElementById('hidden-file-input').click();
        }

        function deleteLocalFile(cid) {
            state.files = state.files.filter(f => f.cid !== cid);
            renderFilesTable();
            showToast("تم إزالة كتلة الملف من الـ BlockStore المحلي", "warning");
        }

        // --- 👥 CONTACTS BOOK SECTION ---
        function renderContactsGrid() {
            const grid = document.getElementById('contacts-grid-list');
            const detailGrid = document.getElementById('contacts-grid-list');
            if (!grid) return;
            grid.innerHTML = "";

            state.contacts.forEach(c => {
                const card = document.createElement('div');
                card.className = "explore-agent-card";
                const isOnline = c.status === 'online';
                
                card.innerHTML = '<div style="font-weight:800; font-size:0.95rem; color:var(--text-main); display:flex; align-items:center; gap:0.5rem;">' + 
                                     '<span style="width:8px; height:8px; border-radius:50%; background:' + (isOnline ? 'var(--accent-emerald)' : 'var(--text-muted)') + '; display:inline-block;"></span>' +
                                     c.nickname + 
                                 '</div>' +
                                 '<div style="font-size:0.7rem; font-family:monospace; color:var(--accent-cyan); overflow:hidden; text-overflow:ellipsis;">' + c.did + '</div>' +
                                 '<div style="font-size:0.75rem; color:var(--text-muted);">Capabilities: ' + c.caps.join(", ") + '</div>' +
                                 '<div style="display:flex; gap:0.5rem; margin-top:0.5rem;">' +
                                     '<button class="btn btn-primary" style="font-size:0.75rem; padding:0.35rem 0.75rem; flex-grow:1;" onclick="openDirectChat(\'' + c.did + '\')">مراسلة</button>' +
                                     '<button class="btn btn-secondary" style="font-size:0.75rem; padding:0.35rem 0.75rem;" onclick="removeContact(\'' + c.did + '\')">إزالة</button>' +
                                 '</div>';
                grid.appendChild(card);
            });

            // Update chats sidebar contacts list as well
            renderChatsSidebar();
        }

        function executeAddContact() {
            const aliasInput = document.getElementById('contact-add-alias');
            const didInput = document.getElementById('contact-add-did');
            const alias = aliasInput.value.trim();
            const did = didInput.value.trim();

            if (!alias || !did) {
                showToast("الرجاء تعبئة كافة الحقول بشكل صحيح", "error");
                return;
            }

            state.contacts.push({
                nickname: alias,
                did,
                status: "offline",
                caps: ["chat:1to1"]
            });

            aliasInput.value = "";
            didInput.value = "";
            renderContactsGrid();
            showToast("تم إضافة جهة الاتصال بنجاح", "success");
        }

        function removeContact(did) {
            state.contacts = state.contacts.filter(c => c.did !== did);
            renderContactsGrid();
            showToast("تم إزالة جهة الاتصال من الدفتر", "warning");
        }

        // --- 🛠️ DEV TOOLS & DOMAINS MANAGEMENT SECTION ---
        async function executeUpdateDomain() {
            const nameInput = document.getElementById('devtools-domain-name');
            const manifestInput = document.getElementById('devtools-domain-manifest');
            const domain = nameInput.value.trim();
            const manifest = manifestInput.value.trim();

            if (!domain || !manifest) {
                showToast("الرجاء تعبئة اسم النطاق ومعرف الـ ManifestCID", "error");
                return;
            }

            showToast("جاري إرسال التزام الحجز (Commit) للشبكة...", "warning");
            
            // Perform actual domain commit
            const commit = await fetchAPI("/api/domain/commit", "POST", { domain });
            if (commit) {
                showToast("تم قبول التزام النطاق بالشبكة بنجاح", "success");
                logAPIRequest("DOMAIN_COMMIT", "Published commitment for domain '" + domain + "' hash=" + commit.hash);
                
                // Add domain count increment
                document.getElementById('home-passport-domain-count').innerText = "1 Domain";
            }
        }

        // ACP Delegation Dialog Actions
        function openNewDelegationModal() {
            document.getElementById('modal-new-delegation').style.display = 'flex';
        }

        function closeNewDelegationModal() {
            document.getElementById('modal-new-delegation').style.display = 'none';
        }

        function executeIssueDelegation() {
            const toDid = document.getElementById('delegation-to-did').value.trim();
            const caps = document.getElementById('delegation-capabilities').value.trim();
            const expires = document.getElementById('delegation-expires').value.trim();

            if (!toDid || !caps) {
                showToast("الرجاء تعبئة كافة الحقول لإصدار التفويض", "error");
                return;
            }

            // Issue mock success
            showToast("تم إصدار تفويض الصلاحيات الرقمي بنجاح", "success");
            logAPIRequest("ACP_DELEGATE", "Issued capabilities delegation to=" + toDid + " caps=" + caps);
            
            // Add sent count
            const count = parseInt(document.getElementById('devtools-delegations-sent-count').innerText) + 1;
            document.getElementById('devtools-delegations-sent-count').innerText = count;

            closeNewDelegationModal();
        }

        // Shared Clipboard Utility
        function copyToClipboardText(text, successMsg) {
            navigator.clipboard.writeText(text);
            showToast(successMsg, "success");
        }

        function showToast(message, type = "success") {
            const container = document.getElementById('toast-container');
            const toast = document.createElement('div');
            toast.className = "toast " + type;
            toast.innerText = message;
            container.appendChild(toast);
            
            setTimeout(() => {
                toast.style.animation = "toastIn 0.3s cubic-bezier(0.16, 1, 0.3, 1) reverse";
                setTimeout(() => toast.remove(), 300);
            }, 3000);
        }
    </script>
</body>
</html>
`
