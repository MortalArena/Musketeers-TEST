# Musketeers Frontend Library Integration

**Document Version:** 1.0  
**Date:** 2025-11-28  
**Phase:** 3.2 - Frontend Library Integration Complete  
**Status:** Complete

---

## Executive Summary

This document specifies the recommended React, TypeScript, and supporting libraries for the Musketeers Wails frontend. The library selection is based on the backend capabilities mapped in the previous document, ensuring optimal integration with the Go backend while maintaining modern frontend best practices.

---

## 1. Core Framework

### 1.1 React

**Library:** `react` (v18.3.x)
**Library:** `react-dom` (v18.3.x)

**Rationale:**
- Industry-standard for building user interfaces
- Excellent TypeScript support
- Large ecosystem and community
- Compatible with Wails v3

**Installation:**
```bash
npm install react react-dom
```

**Type Definitions:**
```bash
npm install --save-dev @types/react @types/react-dom
```

---

### 1.2 Wails

**Library:** `@wailsapp/runtime` (v3.x)

**Rationale:**
- Official Wails runtime for Go-React integration
- Enables Go-to-JavaScript calls
- Provides native window management
- Cross-platform desktop application support

**Installation:**
```bash
npm install @wailsapp/runtime
```

**Usage:**
```typescript
import { EventsOn, EventsEmit } from '@wailsapp/runtime';

// Call Go function
import { GetSessions } from '../../wailsjs/go/main/App';

// Listen to Go events
EventsOn('session-update', (data) => {
    console.log('Session updated:', data);
});
```

---

## 2. State Management

### 2.1 Zustand

**Library:** `zustand` (v4.5.x)

**Rationale:**
- Lightweight and simple API
- Excellent TypeScript support
- No boilerplate required
- Perfect for session, agent, and task state

**Installation:**
```bash
npm install zustand
```

**Store Example:**
```typescript
import { create } from 'zustand';

interface SessionStore {
    sessions: Session[];
    currentSession: Session | null;
    setSessions: (sessions: Session[]) => void;
    setCurrentSession: (session: Session | null) => void;
    addSession: (session: Session) => void;
    removeSession: (sessionId: string) => void;
}

const useSessionStore = create<SessionStore>((set) => ({
    sessions: [],
    currentSession: null,
    setSessions: (sessions) => set({ sessions }),
    setCurrentSession: (session) => set({ currentSession: session }),
    addSession: (session) => set((state) => ({ 
        sessions: [...state.sessions, session] 
    })),
    removeSession: (sessionId) => set((state) => ({
        sessions: state.sessions.filter(s => s.id !== sessionId)
    })),
}));
```

---

### 2.2 React Query

**Library:** `@tanstack/react-query` (v5.x)

**Rationale:**
- Automatic data fetching and caching
- Optimistic updates
- Background refetching
- Perfect for REST API integration

**Installation:**
```bash
npm install @tanstack/react-query
```

**Usage Example:**
```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

// Fetch sessions
function useSessions() {
    return useQuery({
        queryKey: ['sessions'],
        queryFn: async () => {
            const response = await fetch('/sessions');
            return response.json();
        }
    });
}

// Create session
function useCreateSession() {
    const queryClient = useQueryClient();
    
    return useMutation({
        mutationFn: async (data: CreateSessionInput) => {
            const response = await fetch('/sessions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            return response.json();
        },
        onSuccess: (data) => {
            queryClient.invalidateQueries({ queryKey: ['sessions'] });
        }
    });
}
```

---

## 3. Routing

### 3.1 React Router

**Library:** `react-router-dom` (v6.22.x)

**Rationale:**
- Industry-standard routing library
- Excellent TypeScript support
- Nested routes support
- Code splitting capabilities

**Installation:**
```bash
npm install react-router-dom
```

**Route Configuration:**
```typescript
import { createBrowserRouter, RouterProvider } from 'react-router-dom';

const router = createBrowserRouter([
    {
        path: '/',
        element: <Layout />,
        children: [
            { index: true, element: <Dashboard /> },
            { path: 'sessions', element: <SessionList /> },
            { path: 'sessions/:id', element: <SessionDetail /> },
            { path: 'agents', element: <AgentRegistry /> },
            { path: 'artifacts', element: <Artifacts /> },
            { path: 'settings', element: <Settings /> },
        ]
    }
]);

function App() {
    return <RouterProvider router={router} />;
}
```

---

## 4. UI Component Library

### 4.1 shadcn/ui

**Library:** `shadcn-ui` (via Radix UI + Tailwind CSS)

**Rationale:**
- Modern, accessible components
- Built on Radix UI primitives
- Tailwind CSS styling
- Highly customizable
- Excellent TypeScript support

**Installation:**
```bash
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
npm install class-variance-authority clsx tailwind-merge
npm install @radix-ui/react-slot @radix-ui/react-dialog @radix-ui/react-dropdown-menu @radix-ui/react-tabs @radix-ui/react-select
```

**Component Examples:**

#### Button
```typescript
import { cva, type VariantProps } from 'class-variance-authority';

const buttonVariants = cva(
    'inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors',
    {
        variants: {
            variant: {
                default: 'bg-primary text-primary-foreground hover:bg-primary/90',
                destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
                outline: 'border border-input bg-background hover:bg-accent hover:text-accent-foreground',
                secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
                ghost: 'hover:bg-accent hover:text-accent-foreground',
                link: 'text-primary underline-offset-4 hover:underline',
            },
            size: {
                default: 'h-10 px-4 py-2',
                sm: 'h-9 rounded-md px-3',
                lg: 'h-11 rounded-md px-8',
                icon: 'h-10 w-10',
            },
        },
        defaultVariants: {
            variant: 'default',
            size: 'default',
        },
    }
);

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(({ className, variant, size, ...props }, ref) => {
    return (
        <button
            className={cn(buttonVariants({ variant, size, className }))}
            ref={ref}
            {...props}
        />
    );
});
```

#### Dialog
```typescript
import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';

const Dialog = DialogPrimitive.Root;
const DialogTrigger = DialogPrimitive.Trigger;
const DialogPortal = DialogPrimitive.Portal;
const DialogClose = DialogPrimitive.Close;

const DialogContent = React.forwardRef<
    React.ElementRef<typeof DialogPrimitive.Content>,
    React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content>
>(({ className, children, ...props }, ref) => (
    <DialogPortal>
        <DialogOverlay />
        <DialogPrimitive.Content
            ref={ref}
            className={cn(
                'fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200',
                className
            )}
            {...props}
        >
            {children}
        </DialogPrimitive.Content>
    </DialogPortal>
));
```

---

### 4.2 Additional UI Components

**Lucide React** - Icons
```bash
npm install lucide-react
```

**React Hot Toast** - Notifications
```bash
npm install react-hot-toast
```

**Recharts** - Charts for progress visualization
```bash
npm install recharts
```

---

## 5. Styling

### 5.1 Tailwind CSS

**Library:** `tailwindcss` (v3.4.x)

**Rationale:**
- Utility-first CSS framework
- Excellent for rapid development
- Responsive design support
- Dark mode support
- Customizable via config

**Installation:**
```bash
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

**Configuration (tailwind.config.js):**
```javascript
/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './src/**/*.{js,jsx,ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        // ... more colors
      },
    },
  },
  plugins: [],
}
```

---

### 5.2 CSS Variables

**Global CSS (index.css):**
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --primary: 221.2 83.2% 53.3%;
    --primary-foreground: 210 40% 98%;
    --secondary: 210 40% 96.1%;
    --secondary-foreground: 222.2 47.4% 11.2%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;
    --accent: 210 40% 96.1%;
    --accent-foreground: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 210 40% 98%;
    --border: 214.3 31.8% 91.4%;
    --input: 214.3 31.8% 91.4%;
    --ring: 221.2 83.2% 53.3%;
    --radius: 0.5rem;
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --primary: 217.2 91.2% 59.8%;
    --primary-foreground: 222.2 47.4% 11.2%;
    --secondary: 217.2 32.6% 17.5%;
    --secondary-foreground: 210 40% 98%;
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
    --accent: 217.2 32.6% 17.5%;
    --accent-foreground: 210 40% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 210 40% 98%;
    --border: 217.2 32.6% 17.5%;
    --input: 217.2 32.6% 17.5%;
    --ring: 224.3 76.3% 48%;
  }
}
}
```

---

## 6. Forms

### 6.1 React Hook Form

**Library:** `react-hook-form` (v7.51.x)

**Rationale:**
- Performance-optimized form handling
- Excellent TypeScript support
- Minimal re-renders
- Easy validation integration

**Installation:**
```bash
npm install react-hook-form
```

**Usage Example:**
```typescript
import { useForm } from 'react-hook-form';

interface CreateSessionForm {
    title: string;
    description: string;
}

function CreateSessionDialog() {
    const { register, handleSubmit, formState: { errors } } = useForm<CreateSessionForm>();
    
    const onSubmit = async (data: CreateSessionForm) => {
        await createSession(data);
    };
    
    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <input {...register('title', { required: true })} />
            {errors.title && <span>Title is required</span>}
            
            <textarea {...register('description')} />
            
            <button type="submit">Create Session</button>
        </form>
    );
}
```

---

### 6.2 Zod Validation

**Library:** `zod` (v3.22.x)

**Rationale:**
- TypeScript-first schema validation
- Excellent integration with react-hook-form
- Runtime type safety

**Installation:**
```bash
npm install zod
```

**Usage Example:**
```typescript
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';

const createSessionSchema = z.object({
    title: z.string().min(1, 'Title is required'),
    description: z.string().optional(),
});

type CreateSessionForm = z.infer<typeof createSessionSchema>;

function CreateSessionDialog() {
    const { register, handleSubmit, formState: { errors } } = useForm<CreateSessionForm>({
        resolver: zodResolver(createSessionSchema)
    });
    
    // ... rest of component
}
```

---

## 7. Data Fetching

### 7.1 Axios

**Library:** `axios` (v1.6.x)

**Rationale:**
- Promise-based HTTP client
- Request/response interceptors
- Automatic JSON transformation
- Better error handling than fetch

**Installation:**
```bash
npm install axios
```

**API Client Setup:**
```typescript
import axios from 'axios';

const api = axios.create({
    baseURL: 'http://localhost:8080',
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Request interceptor
api.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => Promise.reject(error)
);

// Response interceptor
api.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            // Handle unauthorized
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

export default api;
```

---

## 8. WebSocket

### 8.1 Native WebSocket

**Library:** Native browser WebSocket API

**Rationale:**
- No additional dependencies
- Built-in browser support
- Sufficient for current requirements

**WebSocket Client:**
```typescript
class WebSocketClient {
    private ws: WebSocket | null = null;
    private sessionId: string;
    private reconnectAttempts: number = 0;
    private maxReconnectAttempts: number = 5;
    private eventHandlers: Map<string, (data: any) => void> = new Map();

    constructor(sessionId: string) {
        this.sessionId = sessionId;
    }

    connect(): void {
        const url = `ws://localhost:8080/ws?session_id=${this.sessionId}`;
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
        };

        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const handler = this.eventHandlers.get(data.type);
            if (handler) {
                handler(data.payload);
            }
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.reconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    on(eventType: string, handler: (data: any) => void): void {
        this.eventHandlers.set(eventType, handler);
    }

    off(eventType: string): void {
        this.eventHandlers.delete(eventType);
    }

    private reconnect(): void {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            setTimeout(() => this.connect(), 1000 * this.reconnectAttempts);
        }
    }

    disconnect(): void {
        if (this.ws) {
            this.ws.close();
        }
    }
}
```

---

## 9. Date Handling

### 9.1 date-fns

**Library:** `date-fns` (v3.x)

**Rationale:**
- Lightweight alternative to Moment.js
- Tree-shakeable
- Excellent TypeScript support
- Immutable operations

**Installation:**
```bash
npm install date-fns
```

**Usage Example:**
```typescript
import { format, formatDistanceToNow } from 'date-fns';

function formatDate(date: Date): string {
    return format(date, 'PPP p');
}

function formatRelativeTime(date: Date): string {
    return formatDistanceToNow(date, { addSuffix: true });
}
```

---

## 10. File Upload

### 10.1 react-dropzone

**Library:** `react-dropzone` (v14.x)

**Rationale:**
- Drag-and-drop file upload
- Excellent TypeScript support
- Customizable UI
- Progress tracking

**Installation:**
```bash
npm install react-dropzone
```

**Usage Example:**
```typescript
import { useDropzone } from 'react-dropzone';

function FileUpload({ sessionId }: { sessionId: string }) {
    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop: async (acceptedFiles) => {
            const formData = new FormData();
            formData.append('file', acceptedFiles[0]);
            
            await api.post(`/sessions/${sessionId}/artifacts`, formData, {
                headers: { 'Content-Type': 'multipart/form-data' }
            });
        }
    });

    return (
        <div {...getRootProps()} className={isDragActive ? 'active' : ''}>
            <input {...getInputProps()} />
            <p>Drag & drop files here, or click to select</p>
        </div>
    );
}
```

---

## 11. Code Editor

### 11.1 Monaco Editor

**Library:** `@monaco-editor/react` (v4.6.x)

**Rationale:**
- VS Code's editor
- Excellent TypeScript/JavaScript support
- Syntax highlighting
- Auto-completion

**Installation:**
```bash
npm install @monaco-editor/react
```

**Usage Example:**
```typescript
import Editor from '@monaco-editor/react';

function CodeEditor({ code, onChange }: { code: string; onChange: (value: string) => void }) {
    return (
        <Editor
            height="400px"
            defaultLanguage="go"
            value={code}
            onChange={(value) => onChange(value || '')}
            theme="vs-dark"
            options={{
                minimap: { enabled: false },
                fontSize: 14,
            }}
        />
    );
}
```

---

## 12. Markdown Rendering

### 12.1 react-markdown

**Library:** `react-markdown` (v9.x)

**Rationale:**
- Markdown to React component
- Customizable renderers
- GitHub-flavored markdown support

**Installation:**
```bash
npm install react-markdown
```

**Usage Example:**
```typescript
import ReactMarkdown from 'react-markdown';

function MarkdownViewer({ content }: { content: string }) {
    return (
        <ReactMarkdown className="prose dark:prose-invert max-w-none">
            {content}
        </ReactMarkdown>
    );
}
```

---

## 13. Internationalization

### 13.1 react-i18next

**Library:** `react-i18next` (v14.x)

**Rationale:**
- Industry-standard i18n library
- Excellent TypeScript support
- Namespace support
- Pluralization

**Installation:**
```bash
npm install react-i18next i18next
```

**Configuration:**
```typescript
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

i18n
    .use(initReactI18next)
    .init({
        resources: {
            en: {
                translation: {
                    'dashboard.title': 'Dashboard',
                    'sessions.list': 'Sessions',
                }
            },
            ar: {
                translation: {
                    'dashboard.title': 'لوحة التحكم',
                    'sessions.list': 'الجلسات',
                }
            }
        },
        lng: 'en',
        fallbackLng: 'en',
        interpolation: {
            escapeValue: false
        }
    });

export default i18n;
```

**Usage:**
```typescript
import { useTranslation } from 'react-i18next';

function Dashboard() {
    const { t } = useTranslation();
    
    return (
        <h1>{t('dashboard.title')}</h1>
    );
}
```

---

## 14. Testing

### 14.1 Testing Library

**Library:** `@testing-library/react` (v14.x)
**Library:** `@testing-library/jest-dom` (v6.x)
**Library:** `@testing-library/user-event` (v14.x)

**Rationale:**
- React Testing Library best practices
- User-centric testing
- Excellent TypeScript support

**Installation:**
```bash
npm install -D @testing-library/react @testing-library/jest-dom @testing-library/user-event
```

---

### 14.2 Jest

**Library:** `jest` (v29.x)
**Library:** `@types/jest` (v29.x)

**Rationale:**
- Industry-standard testing framework
- Fast and reliable
- Excellent TypeScript support

**Installation:**
```bash
npm install -D jest @types/jest ts-jest
```

**Configuration (jest.config.js):**
```javascript
module.exports = {
    preset: 'ts-jest',
    testEnvironment: 'jsdom',
    setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts'],
    moduleNameMapper: {
        '^@/(.*)$': '<rootDir>/src/$1',
    },
};
```

---

### 14.3 MSW (Mock Service Worker)

**Library:** `msw` (v2.x)

**Rationale:**
- API mocking for tests
- Network interception
- TypeScript support

**Installation:**
```bash
npm install -D msw
```

---

## 15. Build Tools

### 15.1 Vite

**Library:** `vite` (v5.x)

**Rationale:**
- Fast development server
- Excellent HMR
- Optimized production builds
- Wails v3 supports Vite

**Installation:**
```bash
npm install -D vite @vitejs/plugin-react
```

**Configuration (vite.config.ts):**
```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
    plugins: [react()],
    server: {
        port: 3000,
    },
    build: {
        outDir: 'dist',
        sourcemap: true,
    },
});
```

---

### 15.2 TypeScript

**Library:** `typescript` (v5.x)

**Rationale:**
- Type safety
- Better IDE support
- Catch errors at compile time

**Installation:**
```bash
npm install -D typescript
```

**Configuration (tsconfig.json):**
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

---

## 16. ESLint & Prettier

### 16.1 ESLint

**Library:** `eslint` (v8.x)
**Library:** `@typescript-eslint/eslint-plugin` (v7.x)
**Library:** `@typescript-eslint/parser` (v7.x)
**Library:** `eslint-plugin-react` (v7.x)
**Library:** `eslint-plugin-react-hooks` (v4.x)

**Installation:**
```bash
npm install -D eslint @typescript-eslint/eslint-plugin @typescript-eslint/parser eslint-plugin-react eslint-plugin-react-hooks
```

**Configuration (.eslintrc.json):**
```json
{
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:react-hooks/recommended",
    "plugin:react/recommended"
  ],
  "parser": "@typescript-eslint/parser",
  "plugins": ["@typescript-eslint", "react-hooks"],
  "rules": {
    "react/react-in-jsx-scope": "off",
    "@typescript-eslint/no-explicit-any": "warn"
  },
  "settings": {
    "react": {
      "version": "detect"
    }
  }
}
```

---

### 16.2 Prettier

**Library:** `prettier` (v3.x)

**Installation:**
```bash
npm install -D prettier
```

**Configuration (.prettierrc):**
```json
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2
}
```

---

## 17. Complete package.json

```json
{
  "name": "musketeers-frontend",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext ts,tsx --report-unused-disable-directives --max-warnings 0",
    "test": "jest"
  },
  "dependencies": {
    "@tanstack/react-query": "^5.0.0",
    "@wailsapp/runtime": "^3.0.0",
    "axios": "^1.6.0",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.0.0",
    "date-fns": "^3.0.0",
    "lucide-react": "^0.300.0",
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "react-dropzone": "^14.0.0",
    "react-hook-form": "^7.51.0",
    "react-hot-toast": "^2.4.0",
    "react-i18next": "^14.0.0",
    "react-markdown": "^9.0.0",
    "react-router-dom": "^6.22.0",
    "recharts": "^2.10.0",
    "tailwind-merge": "^2.0.0",
    "zustand": "^4.5.0",
    "zod": "^3.22.0"
  },
  "devDependencies": {
    "@monaco-editor/react": "^4.6.0",
    "@radix-ui/react-dialog": "^1.0.0",
    "@radix-ui/react-dropdown-menu": "^2.0.0",
    "@radix-ui/react-select": "^2.0.0",
    "@radix-ui/react-slot": "^1.0.0",
    "@radix-ui/react-tabs": "^1.0.0",
    "@testing-library/jest-dom": "^6.0.0",
    "@testing-library/react": "^14.0.0",
    "@testing-library/user-event": "^14.0.0",
    "@types/jest": "^29.0.0",
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@typescript-eslint/eslint-plugin": "^7.0.0",
    "@typescript-eslint/parser": "^7.0.0",
    "@vitejs/plugin-react": "^4.0.0",
    "autoprefixer": "^10.0.0",
    "eslint": "^8.0.0",
    "eslint-plugin-react": "^7.0.0",
    "eslint-plugin-react-hooks": "^4.0.0",
    "i18next": "^23.0.0",
    "jest": "^29.0.0",
    "msw": "^2.0.0",
    "postcss": "^8.0.0",
    "prettier": "^3.0.0",
    "tailwindcss": "^3.4.0",
    "ts-jest": "^29.0.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0"
  }
}
```

---

## 18. Component Structure

### 18.1 Recommended Folder Structure

```
src/
├── components/
│   ├── ui/              # shadcn/ui components
│   │   ├── button.tsx
│   │   ├── dialog.tsx
│   │   ├── input.tsx
│   │   └── ...
│   ├── dashboard/
│   │   ├── Dashboard.tsx
│   │   └── ...
│   ├── sessions/
│   │   ├── SessionList.tsx
│   │   ├── SessionDetail.tsx
│   │   └── ...
│   ├── chat/
│   │   ├── ChatInterface.tsx
│   │   └── ...
│   └── ...
├── hooks/
│   ├── useWebSocket.ts
│   ├── useSessions.ts
│   └── ...
├── lib/
│   ├── api.ts
│   ├── websocket.ts
│   └── utils.ts
├── stores/
│   ├── sessionStore.ts
│   ├── agentStore.ts
│   └── ...
├── types/
│   ├── session.ts
│   ├── agent.ts
│   └── ...
├── App.tsx
├── main.tsx
└── vite-env.d.ts
```

---

## 19. Integration with Wails

### 19.1 wailsjs Directory

Wails automatically generates bindings in `wailsjs/go/`:

```
wailsjs/
└── go/
    └── main/
        ├── App.js
        └── App.d.ts
```

### 19.2 Go Function Calls

**Backend (Go):**
```go
// main.go
package main

import (
    "context"
    
    "github.com/wailsapp/wails/v2/pkg/options"
)

type App struct {
    ctx context.Context
}

func NewApp() *App {
    return &App{}
}

func (a *App) GetSessions() ([]Session, error) {
    // Implementation
    return sessions, nil
}
```

**Frontend (TypeScript):**
```typescript
import { GetSessions } from '../../wailsjs/go/main/App';

async function loadSessions() {
    const sessions = await GetSessions();
    console.log(sessions);
}
```

---

## 20. Performance Optimization

### 20.1 Code Splitting

**Route-based splitting:**
```typescript
const SessionDetail = lazy(() => import('./components/sessions/SessionDetail'));

function App() {
    return (
        <Suspense fallback={<Loading />}>
            <Routes>
                <Route path="/sessions/:id" element={<SessionDetail />} />
            </Routes>
        </Suspense>
    );
}
```

### 20.2 Memoization

**React.memo:**
```typescript
const SessionCard = React.memo(({ session }: { session: Session }) => {
    return <div>{session.title}</div>;
});
```

**useMemo:**
```typescript
const filteredSessions = useMemo(() => {
    return sessions.filter(s => s.status === 'active');
}, [sessions]);
```

---

## 21. Accessibility

### 21.1 ARIA Attributes

**Example:**
```typescript
<button
    aria-label="Close dialog"
    onClick={onClose}
>
    <X className="h-4 w-4" />
</button>
```

### 21.2 Keyboard Navigation

**Example:**
```typescript
<div
    role="button"
    tabIndex={0}
    onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
            onClick();
        }
    }}
    onClick={onClick}
>
    Clickable div
</div>
```

---

## 22. Conclusion

The recommended frontend library stack for Musketeers includes:

**Core:**
- React 18.3.x - UI framework
- Wails v3 - Go-React integration
- TypeScript 5.x - Type safety

**State Management:**
- Zustand 4.5.x - Global state
- React Query 5.x - Server state

**Routing:**
- React Router 6.22.x - Navigation

**UI Components:**
- shadcn/ui - Component library
- Tailwind CSS 3.4.x - Styling
- Lucide React - Icons

**Forms:**
- React Hook Form 7.51.x - Form handling
- Zod 3.22.x - Validation

**Data Fetching:**
- Axios 1.6.x - HTTP client
- Native WebSocket - Real-time

**Utilities:**
- date-fns 3.x - Date handling
- react-dropzone 14.x - File upload
- Monaco Editor 4.6.x - Code editing
- react-markdown 9.x - Markdown rendering
- react-i18next 14.x - Internationalization

**Testing:**
- Jest 29.x - Testing framework
- React Testing Library 14.x - Component testing
- MSW 2.x - API mocking

**Build Tools:**
- Vite 5.x - Build tool
- ESLint 8.x - Linting
- Prettier 3.x - Formatting

This stack provides a **modern, type-safe, and performant** foundation for the Musketeers frontend with excellent integration with the Go backend.

---

**Document End**
