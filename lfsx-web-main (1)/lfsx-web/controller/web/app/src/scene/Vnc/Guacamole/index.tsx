import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react'
import { useCustomizations } from '../../../provider/CustomizationProvider'
import Gua from "guacamole-common-js";
import './index.css'
import SecurityHelper from '../../../services/SecuriyHelper';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
import Keyboard from '../../../components/NoVNC/core/input/keyboard.js'
import LoadingAnimation from '../../../components/LoadingAnimation';

const Guacamole: React.ForwardRefRenderFunction<GuacamoleHandler, GuacamoleProps> = (props, ref) => {

	// Refs for guacamole 
	const [ isLoading, setLoading ] = useState(true)
	const displayRef = useRef<HTMLElement|null>(null);
	const guaRef = useRef<Gua.Client>();
	const stateRef = useRef<number>(5)

	// Customizations settings
	const customizations = useCustomizations()

	// Focuses Guacamole Client Display element if it's parent element has been clicked,
	// because div with GuacamoleClient inside otherwise does not focus.
	const parentOnClickHandler = () => {
		displayRef.current?.focus();
	};

	// NoVNC keyboard
	const vncKeyboard = useRef<any>()

	// Register events
	useEffect(() => {
		let stillValid = true

		// The keyboard does not work as expected in the browser. So we wrap the KeyBoard around the NoVNC keyboard so that it's working
		// normally again
		const keyboard = new Keyboard(window)
		keyboard.grab()

		// Handle events
		keyboard.onkeyevent = (keysym: number, code: string, down: boolean) => {
			if (guaRef.current && stillValid && props.onKeyType(keysym, code, down)) {
				guaRef.current.sendKeyEvent(down ? 1 : 0, keysym)
			} 
		}
		vncKeyboard.current = keyboard

		// "Remove" event listeners in dev mode on rerender
		return () => {
			stillValid = false
			keyboard.ungrab()
		}
	}, [ ])
	
	useImperativeHandle(ref, () => ({

		connect() {
			// We don't want to reconnect again if we are still connected to guacamole
			if (stateRef.current <= 3) {
				console.log("Not trying to connect to guacamole. We already have a connection in status " + stateRef.current)
				return
			}

			const tunnel = new Gua.WebSocketTunnel(props.url)
			tunnel.receiveTimeout = 50000
	
			guaRef.current = new Gua.Client(tunnel)
	
			// Add connection to display
			while (displayRef.current?.firstChild && displayRef.current?.lastChild) {
				displayRef.current?.removeChild(displayRef.current?.lastChild)
			}
			displayRef.current?.appendChild(guaRef.current.getDisplay().getElement())
			displayRef.current?.focus()
	
			// Error handler
			tunnel.onerror = (error: Gua.Status) => {
				stateRef.current = 5
				props.onSocketClose({ code: error.code, reason: error.message ?? "Error", wasClean: true } as CloseEvent)
				console.log("Tunnel closed: " + error.message)
			}

			const handleServerClipboardChange = (stream: any, mimetype: any) => {
				if (mimetype === "text/plain") {
					// stream.onblob = (data) => copyToClipboard(atob(data));
					stream.onblob = (data: any) => {
						const serverClipboard = atob(data);
						// we don't want action if our knowledge of server cliboard is unchanged
						// and also don't want to fire if we just selected several space character accidentaly
						// which hapens often in SSH session
						if (serverClipboard.trim() !== "") {
							// Put data received form server to client's clipboard
							const text = convertFromGuacamole(serverClipboard)
							console.log("Received server clipboard: " + text)
							navigator.clipboard.writeText(text);
						}
					}
				} else {
					// Haven't seen those yet...
					console.log("Unsupported mime type:" + mimetype)
				}
			};
	
			// Add handler only when navigator clipboard is available
			if (navigator.clipboard) {
				guaRef.current.onclipboard = handleServerClipboardChange;
			}
	
			// Set status based on guacamole status
			guaRef.current.onstatechange = function (s: Gua.Client.State) {
				stateRef.current = s

				if (s === 3) {
					console.log("Gaucamole tunnel is connected")
					setLoading(false)
					props.onConnect()

					/* We already have a global event listenere which works better and more consistent */
					// const keyboard = new Gua.Keyboard(displayRef.current!);
					//keyboard.onkeydown = function (keysym) {
					//if (guaRef.current && props.onKeyType(keysym, "Unknwon", true)) guaRef.current.sendKeyEvent(1, fixKeys(keysym));
					//};
					//keyboard.onkeyup = function (keysym) {
					//if (guaRef.current && props.onKeyType(keysym, "Unknwon", false)) guaRef.current.sendKeyEvent(0, fixKeys(keysym));
					//};
			
					// Mouse
					// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
					const mouse = new Gua.Mouse(displayRef.current!);
			
					// These functions does exists. The types package is incorrect!
					// eslint-disable-next-line @typescript-eslint/ban-ts-comment
					/* @ts-ignore */
					mouse.onmousemove = function (mouseState) {
						if (guaRef.current) guaRef.current.sendMouseState(mouseState);
						props.onMouseMove(mouseState)
					};
					// eslint-disable-next-line @typescript-eslint/ban-ts-comment
					/* @ts-ignore */
					mouse.onmousedown = mouse.onmouseup = function (mouseState) {
						if (guaRef.current) guaRef.current.sendMouseState(mouseState);
					};
					/* tslint:enable */

					if (guaRef.current) {
					// Also update locale cursor when the remote cursor changes.
					// This does NOT work correctly. WayVNC doesn't send any messages
					// on ncursor change!
						guaRef.current.getDisplay().showCursor(false)
						guaRef.current.getDisplay().oncursor = (c: HTMLCanvasElement, x: number, y: number) => {
							console.log("Cursor changed")
							mouse.setCursor(c, x, y)
						}
					}
				} else if (s > 3) {
					console.log("Changed guacamole state to " + s)

					// The server closed the connection cleanly (it probably received an invalid instruction from VNC server)
					if (s === 5) {
						props.onSocketClose({ code: 5, reason: "Server closed connection", wasClean: true } as CloseEvent)
					}
				}
			}

			guaRef.current?.connect(
				"scheme=vnc&useGuacamole=true&userIdentifier=" + encodeURIComponent(SecurityHelper.getUserIdentification())
				+ "&quality=" + customizations.quality 
				+ "&scale=" + customizations.scalingFactor
			)
		},

		disconnect()  {
			if (guaRef.current !== undefined) {
				guaRef.current.disconnect()
			}
		},

		sendKey(keysym, code, down) {
			if (guaRef.current) guaRef.current.sendKeyEvent(down ? 1 : 0, keysym)
		},

		clipboardPaste(text) {
			if (guaRef.current) {
				const stream = guaRef.current.createClipboardStream("text/plain");
				const writer = new Gua.StringWriter(stream)
				console.log("Sending clipboard text: " + text)

				writer.onack = () => {
					writer.sendEnd()
					stream.sendEnd()
				}

				writer.sendText(text)
				writer.sendEnd()
			}
		},

		focus() {
			displayRef.current?.focus()
		},

		grabKeyboard() {
			vncKeyboard.current.grab()
		},

		ungrabKeyboard() {
			vncKeyboard.current.ungrab()
		}

	}))

	return (
		<>
			{isLoading && props.disconnectReason === null && <div className="loading-wrapper"> <LoadingAnimation text='Anwendung wird geladen' /></div> }
			{ props.disconnectReason !== null && <div style={{ backgroundColor: "#e8e6e6" }}> {props.disconnectReason.message} </div> }
			<div 
				id="gua-display" 
				className={props.className} 
				ref={displayRef as any} 
				style={{ width: "100%", height: "100%", position: "absolute", backgroundColor: "#e8e6e6" }}
				onClick={parentOnClickHandler}
			>
			</div>
		</>

	)
}

export type GuacamoleProps = {
	url: string
	className: string
	ref: React.MutableRefObject<Gua.Client | undefined>
	onSocketClose: (e: CloseEvent) => void
	disconnectReason: { code: "USER_ALREADY_EXISTS" | "UNKNOWN", message: string  } | null
	onConnect: () => void

	onKeyType: (keysym: number, desc: string, down: boolean) => boolean
	onMouseMove: (e: {x: number, y: number}) => void
}

export type GuacamoleHandler = {
	connect: () => void
	disconnect: () => void
	sendKey: (keysym: number, code: string, down?: boolean) => void
	clipboardPaste: (text: string) => void
	focus: () => void

	grabKeyboard: () => void
	ungrabKeyboard: () => void
}

export function instanceOfGuacamoleHandler(object: any): object is GuacamoleHandler {
	return 'grabKeyboard' in object
}

export default forwardRef(Guacamole)

function convertFromGuacamole(str: string): string {
	return str
		.replaceAll("\u00C3\u0083\u00C2\u00A4", "ä")
		.replaceAll("\u00C3\u0083\u00C2\u0084", "Ä")
		.replaceAll("\u00C3\u0083\u00C2\u00BC", "ü")
		.replaceAll("\u00C3\u0083\u00C2\u009C", "Ü")
		.replaceAll("\u00C3\u0083\u00C2\u0096", "Ö")
		.replaceAll("\u00C3\u0083\u00C2\u00B6", "ö")
		.replaceAll("\u00C3\u0083\u00C2\u009F", "ß")
}