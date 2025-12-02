// updateHandlers.ts
import { servers, serverPools, configs } from "$lib/store";
import {
    Type,
    Status,
} from "../grpc/frontcontrol_pb";
import type { UpdateDataUserResponse } from "../grpc/frontcontrol_pb";
import type { Writable } from "svelte/store";

// ======================================================================
// Mapping Type → Store
// ======================================================================

const storeMap: Record<Type, Writable<any[]> | undefined> = {
    [Type.TYPE_UNKNOWN]: undefined,
    [Type.SERVERPOOL]: serverPools,
    [Type.SERVER]: servers,
    [Type.CONFIG]: configs,
};

// ======================================================================
// Convertit map<string,string> → object JS
// ======================================================================

function mapToObject(map: Record<string, string>): any {
    return { ...map };
}

// ======================================================================
// Vérifie la présence de la clé composite user_id + name
// ======================================================================

function hasCompositeKey(obj: any): boolean {
    if (!obj.user_id || !obj.name) {
        console.warn("Objet sans user_id ou name → ignoré :", obj);
        return false;
    }
    return true;
}

// ======================================================================
// Comparaison des clés composées
// ======================================================================

function isSameKey(a: any, b: any): boolean {
    return a.user_id === b.user_id && a.name === b.name;
}

// ======================================================================
// CREATE / UPDATE / DELETE sur un store avec clé composite
// ======================================================================

function applyStoreMutation(
    store: Writable<any[]>,
    status: Status,
    newObj: any
) {
    if (!hasCompositeKey(newObj)) return;

    store.update(items => {
        if (!Array.isArray(items)) {
            items = [];
        }

        const idx = items.findIndex(i => isSameKey(i, newObj));

        switch (status) {
            case Status.CREATE:
                if (idx === -1) items.push(newObj);
                break;

            case Status.UPDATE:
                if (idx !== -1)
                    items[idx] = { ...items[idx], ...newObj };
                break;

            case Status.DELETE:
                if (idx !== -1)
                    items.splice(idx, 1);
                break;
        }

        return [...items]; // reactive pour Svelte
    });
}

// ======================================================================
// Handler principal
// ======================================================================

export function handleUserUpdate(update: UpdateDataUserResponse) {
    console.log("updating something : ", update)
    const store = storeMap[update.type];

    if (!store) {
        console.warn("Type non géré dans le storeMap:", update.type);
        return;
    }

    const obj = mapToObject(update.data);

    console.log("Store BEFORE mutation:", configs);
    applyStoreMutation(store, update.status, obj);
    console.log("Store AFTER mutation:", configs);
}
