// src/stores/fetchServerpools.ts
import { writable, get } from 'svelte/store';
import type { Writable } from 'svelte/store';
import { authStore } from './authStore';

// Types
export interface User {
  id: string;
  name: string;
  email: string;
}

export interface Serverpool {
  serverpool_id: string;
  image_ref: string;
  flavor_ref: string;
  networks: string[];
  min_vm: number;
  max_vm: number;
  pending_jobs: number;
}

export interface Server {
  id: string;
  name: string;
  status: string;
  flavor_id: string;
  image_id: string;
  addresses: Record<string, { addr: string }[]>;
  created: string;
  updated?: string;
  host_id?: string;
  progress?: number;
}

interface ServerpoolStore {
  user: User | null;
  serverpools: Serverpool[];
  error: string | null;
}

function createServerpoolStore() {
  const { subscribe, set, update }: Writable<ServerpoolStore> = writable({
    user: null,
    serverpools: [],
    error: null
  });

  async function fetchServerpools() {
    const token = get(authStore);
    try {
      // Récupère le profil utilisateur
      const res = await fetch('http://localhost:8080/users/me', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) {
        update(state => ({ ...state, error: "Impossible de récupérer le profil" }));
        return;
      }
      const data = await res.json();
      update(state => ({
        ...state,
        user: { id: data.id, name: data.name, email: data.email },
        error: null
      }));

      // Récupère les serverpools de l'utilisateur
      const res2 = await fetch('http://localhost:8080/serverpool/mysp', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res2.ok) {
        update(state => ({ ...state, error: "Impossible de récupérer les serverpools" }));
        return;
      }
      const data2 = await res2.json();
      update(state => ({
        ...state,
        serverpools: data2.serverpools || [],
        error: null
      }));
    } catch (err) {
      update(state => ({ ...state, error: "Erreur backend" }));
      console.error(err);
    }
  }

  async function fetchServersInServerpool(serverpoolId: string): Promise<Server[]> {
    const token = get(authStore);
    try {
      // Correction de l'URL pour la bonne route backend
      const res = await fetch(`http://localhost:8080/serverpool/mysp/${serverpoolId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) {
        throw new Error("Impossible de récupérer les serveurs du serverpool");
      }
      const data = await res.json();
      return data.servers || [];
    } catch (err) {
      console.error(err);
      return [];
    }
  }

  return {
    subscribe,
    fetchServerpools,
    fetchServersInServerpool,
  };
}

// Création d'un serverpool
async function createServerpool(serverpool: {
  namesp: string;
  image_ref: string;
  flavor_ref: string;
  networks: string[];
  min_vm: number;
  max_vm: number;
}) {
  const token = get(authStore);
  try {
    
    const res = await fetch(`http://localhost:8080/serverpool`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(serverpool)
    });
    if (!res.ok) {
      throw new Error("Impossible de créer le serverpool");
    } else {
      // Actualiser la liste des serverpools après la création
      await serverpoolStore.fetchServerpools();
    }
  } catch (err) {
    console.error(err);
    throw err;
  }
}

export interface ImageOption {
  value: string;
  name: string;
  status: string;
  // Ajoute d'autres champs si besoin
}

async function fetchAllImages(): Promise<ImageOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/images', {
      headers: { 'Authorization': `Bearer ${token}` } // Ajout du token d'authentification
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les images");
    }
    const data = await res.json();
    return (data.images || []).map((img: any) => ({
      value: img.id,
      name: img.name || img.id,
      status: img.status
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

export interface FlavorOption {
  value: string;
  name: string;
}

async function fetchAllFlavors(): Promise<FlavorOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/flavor', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les flavors");
    }
    const data = await res.json();
    return (data.flavors || []).map((flavor: any) => ({
      value: flavor.id,
      name: flavor.name || flavor.id
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

export interface NetworkOption {
  value: string;
  name: string;
}

async function fetchAllNetworks(): Promise<NetworkOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/networks', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les réseaux");
    }
    const data = await res.json();
    return (data.networks || []).map((net: any) => ({
      value: net.id,
      name: net.name || net.id
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

// Exports

export { createServerpool, fetchAllImages, fetchAllFlavors, fetchAllNetworks };
export const serverpoolStore = createServerpoolStore();
