import { writable } from 'svelte/store';
import { jwtDecode } from 'jwt-decode';
import { goto } from '$app/navigation';
import { authenticateUser } from '$lib/grpc/authService/authService';
import { subscribeUserUpdate } from '$lib/grpc/userUpdateService/userService';
import { resetAll } from './serverpoolStore';

interface JwtPayload {
  exp: number;
  [key: string]: any;
}

interface AuthData {
  token: string;
  email: string;
}

function createAuthStore() {
  let initial: AuthData | null = null;

  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('authData');
    if (saved) {
      const data: AuthData = JSON.parse(saved);
      if (isTokenValid(data.token)) {
        initial = data;
      } else {
        localStorage.removeItem('authData');
      }
    }
  }

  const store = writable<AuthData | null>(initial);

  // Persist state
  store.subscribe((auth) => {
    if (typeof window === 'undefined') return;


    if (auth)
      localStorage.setItem('authData', JSON.stringify(auth));
    else
      localStorage.removeItem('authData');
  });

  return store;
}

export const authStore = createAuthStore();


// ---------------------------
// Helpers
// ---------------------------

function isTokenValid(token: string) {
  try {
    const decoded = jwtDecode<JwtPayload>(token);
    return decoded.exp > Date.now() / 1000;
  } catch {
    return false;
  }
}

// ---------------------------
// Login / Logout
// ---------------------------

export function login(token: string, email: string) {
  authStore.set({ token, email });
}

export function logout() {
  authStore.set(null);
  resetAll();
  goto("/");
}

export async function tryLogin(email: string, password: string) {
  if (!email || !password) {
    return { success: false, error: 'Champs non rempli' };
  }

  try {
    const result = await authenticateUser(email, password);

    if (!result.success || !result.token) {
      return { success: false, error: 'Erreur lors de la connexion' };
    }

    login(result.token, email);

    return { success: true };
  } catch (err) {
    console.error(err);
    return { success: false, error: 'Erreur backend' };
  }
}
