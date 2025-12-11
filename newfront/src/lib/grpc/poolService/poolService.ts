import { createClient } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { create } from "@bufbuild/protobuf"

import {
    PoolService,
    CreatePoolRequestSchema,
    GetPoolRequestSchema,
    DeletePoolRequestSchema,
    RebuildServerRequestSchema,
} from "../frontcontrol_pb"

import type {
    CreatePoolResponse,
    GetPoolResponse,
    DeletePoolResponse,
    RebuildServerResponse,
    CreatePoolRequest,
    GetPoolRequest,
    DeletePoolRequest,
    RebuildServerRequest,
} from "../frontcontrol_pb"

const transport = createGrpcWebTransport({
    baseUrl: "http://localhost:80",
    useBinaryFormat: true,
    interceptors: [],
    fetch: globalThis.fetch,
    jsonOptions: {},
});

const poolClient = createClient(PoolService, transport);

export async function createPool(req: CreatePoolRequest): Promise<CreatePoolResponse> {
    try {
        const res: CreatePoolResponse = await poolClient.createPool(req);
        return res;
    } catch (err) { 
        console.error("Erreur creation d'un pool :", err);
        throw err;
    }
}

export async function getPool(req: GetPoolRequest): Promise<GetPoolResponse> {
    try {
        const res: GetPoolResponse = await poolClient.getPool(req);
        return res;
    } catch (err) { 
        console.error("Erreur recuperation d'un pool :", err);
        throw err;
    }
}

export async function deletePool(req: DeletePoolRequest): Promise<DeletePoolResponse> {
    try {
        const res: DeletePoolResponse = await poolClient.deletePool(req);
        return res;
    } catch (err) { 
        console.error("Erreur delete d'un pool :", err);
        throw err;
    }
}

export async function rebuildServer (req: RebuildServerRequest): Promise<RebuildServerResponse> {
    try {
        const res: RebuildServerResponse = await poolClient.rebuildServer(req);
        return res;
    } catch (err) {
        console.error("Error rebuilding server: ", err)
        throw err;
    }
}

export async function addServer (req: CreatePoolRequest): Promise<RebuildServerResponse> {
    try {
        const res: RebuildServerResponse = await poolClient.addServer(req);
        return res;
    } catch (err) {
        console.error("Error adding server: ", err)
        throw err;
    }
}
