
let socket: WebSocket | null = null;
let reconnectTimeout: NodeJS.Timeout | null = null;
let isManuallyClosed = false;

export function connectWebSocket(token: string) {
	console.log("🔌 Tentative de connexion WebSocket...");

	if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
    console.log("⚠️ Tentative de double connexion WebSocket ignorée");
    return socket;
  }

	socket = new WebSocket(`ws://localhost:8080/ws?token=${token}`); // ou wss:// en prod

	socket.onopen = () => {
		console.log("🟢 WebSocket connecté");
	};

	socket.onmessage = (event) => {
		console.log("📩 Message reçu :", event.data);
		// → Tu peux ici dispatcher des événements ou mettre à jour un store Svelte
	};

	socket.onclose = () => {
        if (token && !reconnectTimeout && !isManuallyClosed) {
            console.log("🔴 WebSocket fermé. Tentative de reconnexion...");
			reconnectTimeout = setTimeout(() => {
				connectWebSocket(token);
				reconnectTimeout = null;
			}, 3000);
		}
	};

	socket.onerror = (err) => {
		console.error("⚠️ Erreur WebSocket :", err);
		socket?.close();
	};

	return socket;
}

export function disconnectWebSocket() {
    isManuallyClosed = true;
    if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
        reconnectTimeout = null;
    }
	if (socket) {
		socket.close();
		socket = null;
		console.log("👋 WebSocket déconnecté");
	}
}
