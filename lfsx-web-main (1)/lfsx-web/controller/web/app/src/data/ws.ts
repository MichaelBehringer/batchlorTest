import { FileUploadRequest } from "./file"


export type WebSocketData = {

	/** Unique id of the data */
	id: number

	/** Used for basic "request/response" mechanism: contains the ID of the responding data */
	responseTo: number | null

	/** Messages containing the real data */
	messages: Array<WebSocketMessage>
}

export type WebSocketMessage = {

	// The type of the message
	type: "LoginRequest" | "LfsStartup" | "Stop" | "OpenInBrowser" | "FileUploadRequest" | "FileUploadFinished"

	// One of the following types as the message data
	openInBrowser?: OpenInBrowser 
	fileUploadRequest?: FileUploadRequest
}

export type OpenInBrowser = {
	url: string
}