// src/stores/fetchServerpools.ts
import { get } from 'svelte/store';
import { authStore } from './authStore';


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
    }
  } catch (err) {
    console.error(err);
    throw err;
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
      console.log("Serveur reconstruit avec succès");
    }
  } catch (err) {
    console.error(err);
    throw err;
  } 
}

export { createServerpool, deleteServerpool, rebuildServer };

