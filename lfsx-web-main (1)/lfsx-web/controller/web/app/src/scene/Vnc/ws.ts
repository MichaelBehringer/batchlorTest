import { WebSocketData, WebSocketMessage } from "../../data/ws";
import SecurityHelper from "../../services/SecuriyHelper";

let ws: WebSocket | null = null;

export function connect(onMessage: (id: number, responseTo: number | null, message: WebSocketMessage) => void) {

	const wsUrl = (location.protocol === 'https:' ? 'wss' : 'ws') + '://'  + location.host + "/api/app/ws?userIdentifier=" + encodeURIComponent(SecurityHelper.getUserIdentification())
	ws = new WebSocket(wsUrl);

	ws.onopen = function() {
		console.log("Openend WebSocket to the LFS.X host")
	};
  
	ws.onmessage = function(e: MessageEvent) {
		const data = JSON.parse(e.data) as WebSocketData

		data.messages.forEach(m => {
			onMessage(data.id, data.responseTo, m)
		})
	};
  
	ws.onclose = function(e) {
		console.log('WebSocket to the LFS.X was closed. Reconnecting in 2 seconds', e.reason);
		ws = null;
		setTimeout(function() {
			connect(onMessage);
		}, 2000);
	};
  
	ws.onerror = function(err: any) {
		console.error('WebSocket to the LFS.X encountered error: ', err.message, 'Closing socket');
		if (ws !== null) ws.close();
		ws = null;
	};
}

export function send(responseTo: number | null, ...messages: Array<WebSocketMessage>) {
	const data: WebSocketData = {
		id: Math.floor(Math.random() * 1048576) + 1,
		messages: messages,
		responseTo: responseTo
	}

	if (ws != null) {
		ws.send(JSON.stringify(data))
	}
}