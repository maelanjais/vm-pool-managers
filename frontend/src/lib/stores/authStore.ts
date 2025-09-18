import { writable } from 'svelte/store';

export const authStore = writable<string | null>(null);

// Initialisation côté client seulement
if (typeof window !== 'undefined') {
  const token = localStorage.getItem('authToken');
  authStore.set(token);
}
