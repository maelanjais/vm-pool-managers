import { AuthServiceClient } from "$lib/grpc/FrontcontrolServiceClientPb";
import * as pb from '../grpc/frontcontrol_pb';

const client = new AuthServiceClient('http://localhost:8080');

export async function CreateUser(
    username: string,
    password: string,
    email: string)
    : Promise<{ userId: string, success: boolean }> {
    const req = new pb.CreateUserRequest();
    req.setUsername(username);
    req.setPassword(password);
    req.setEmail(email);

    try {
        const resp = await client.createUser(req, null);
        const userId = resp.getUserId();
        return { userId, success: true };
    } catch (error) {
        console.error('Error creating user:', error);
        return { userId: '', success: false };
    }
}

export async function AuthenticateUser (
    email: string,
    password: string)
    : Promise<{ token: string, success: boolean }> {
    const req = new pb.AuthenticateUserRequest();
    req.setEmail(email);
    req.setPassword(password);

    try {
        const resp = await client.authenticateUser(req, null);
        const token = resp.getToken();
        return { token, success: true };
    } catch (error) {
        console.error('Error authenticating user:', error);
        return { token: '', success: false };
    }
}

export function loginUser(token: string) {
    localStorage.setItem('authToken', token);
    
}