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
  flavor: { id: string; name: string | null };
  image: { id: string; name: string | null };
  addresses: Record<string, { addr: string }[]>;
  created: string;
  updated?: string;
  host_id?: string;
  progress?: number;
}

export interface ImageOption {
  value: string;
  name: string;
  status: string;
  Mindisk: number;
  Minram: number;
}

export type ImageGroupe = Record<string, ImageOption[]>;
export interface GroupedImageOption {
  group: string;   // la clé du groupe, ex: "sl" ou "ubuntu"
  value: string;   // l’ID de l’image
  name: string;
  status: string;
  Mindisk: number;
  Minram: number;
}

export interface GroupeImageName {
  name: string;
  value: string;
}

export interface FlavorOption {
  value: string;
  name: string;
  disk: number;
  ram: number;
  vcpus: number;
  rxtx_factor: number;
}

export interface NetworkOption {
  value: string;
  name: string;
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
    return data.map((img: any) => ({
      value: img.id,
      name: img.name || img.id,
      status: img.status,
      Mindisk: img.min_disk,
      Minram: img.min_ram
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
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
    return data.map((flavor: any) => ({
      value: flavor.id,
      name: flavor.name || flavor.id,
      disk: flavor.disk,
      ram: flavor.ram,
      vcpus: flavor.vcpus,
      rxtx_factor: flavor.rxtx_factor
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
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
    return data.map((net: any) => ({
      value: net.id,
      name: net.name || net.id,
      disk: net.disk,
      ram: net.ram,
      vcpus: net.vcpus,
      rxtx_factor: net.rxtx_factor
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

// Supprimer un serverpool
async function deleteServerpool(serverpoolId: string) {
  const token = get(authStore);
  try {
    const res = await fetch(`http://localhost:8080/serverpool/${serverpoolId}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de supprimer le serverpool");
    } else {
      // Actualiser la liste des serverpools après la suppression
      await serverpoolStore.fetchServerpools();
    }
  } catch (err) {
    console.error(err);
    throw err;
  }
}

async function rebuildServer(serverId: string, serverName: string, imageId: string) {
  const token = get(authStore);
  try {
    const res = await fetch(`http://localhost:8080/serverpool/rebuild`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({ serverId: serverId, server_name: serverName, image_id: imageId })
    });
    if (!res.ok) {
      throw new Error("Impossible de reconstruire le serveur");
    } else {
      // Optionnel : Actualiser les données du serveur après la reconstruction
      console.log("Serveur reconstruit avec succès");
    }
  } catch (err) {
    console.error(err);
    throw err;
  } 
}

async function fetchGroupImages(group :string): Promise<ImageOption[]> {
  const token = get(authStore); // récupère le token

  try {
    const res = await fetch('http://localhost:8080/serverpool/imagegroup', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({group})
    });

    if (!res.ok) {
      throw new Error("Impossible de récupérer les images du groupe");
    }

    const data = await res.json();
    return data.map((img: any) => ({
      value: img.id,
      name: img.name || img.id,
      status: img.status,
      Mindisk: img.min_disk,
      Minram: img.min_ram
    }));
  } catch (err){
    console.error(err);
    return [];
  }
}



async function fetchGroupImageName(): Promise<GroupeImageName[]> {
  const token = get(authStore);

  try {
    const res = await fetch(`http://localhost:8080/serverpool/groupimagesname`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`
      },
    });

    if (!res.ok) {
      throw new Error("Impossible de récupérer les images groupées");
    }

    const data: GroupeImageName[] = await res.json();
    return data;
  } catch (err) {
    console.error(err);
    return [];
  }
}

// Exports

export { createServerpool, fetchAllImages, fetchAllFlavors, fetchAllNetworks, deleteServerpool, rebuildServer , fetchGroupImages , fetchGroupImageName};
export const serverpoolStore = createServerpoolStore();
