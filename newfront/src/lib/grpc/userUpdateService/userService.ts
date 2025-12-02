import { createClient } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { create } from "@bufbuild/protobuf";

import {
    UserService,
    UpdateDataUserRequestSchema,
    UpdateDataUserResponseSchema,
} from "../frontcontrol_pb"

import type { 
    UpdateDataUserRequest,
    UpdateDataUserResponse,
} from "../frontcontrol_pb"
import { handleUserUpdate } from "$lib/utils/updateHandlers";


const transport = createGrpcWebTransport({
  baseUrl: "http://localhost:80", // l'URL de ton proxy gRPC-Web
  useBinaryFormat: true,             // recommandé pour gRPC-Web
  interceptors: [],                  // tu peux ajouter des middlewares si besoin
  fetch: globalThis.fetch,           // le fetch du navigateur
  jsonOptions: {},                   // options pour JSON, si tu veux
});

const userclient = createClient(UserService, transport);

export async function subscribeUserUpdate(user: string) {
    const req = create(UpdateDataUserRequestSchema, {user});
    console.log("Envoi request stream :", req);
    const stream = userclient.updateDataUser(req);

    try {
        for await(const update of stream) { 
            handleUserUpdate(update)
        }
    } catch (err) {
        console.error("Erreur stream UserService:", err);
    }
}

