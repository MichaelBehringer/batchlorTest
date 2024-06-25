import React, { useEffect, useRef, useState } from 'react'
import './index.css'
import VncScreen, { VncScreenHandle }  from '../../components/VncScreen';
import LoadingAnimation from '../../components/LoadingAnimation';
import { RequestHelper, StandardResponse } from '../../services/RequestService';
import { getItems, hasItemChanged, toogleFullscreen } from './toolbar';
import { probe, resizeWindow, scaleWindowHot } from '../../data/vnc';
import { WebSocketMessage } from '../../data/ws';
import { connect, send } from './ws';
import { useNavigate } from 'react-router-dom';
import { doLogout } from '../../data/login';
import { VncSettings } from './Settings';
import { useCustomizations } from '../../provider/CustomizationProvider';
import SecurityHelper from '../../services/SecuriyHelper';
import Guacamole, { GuacamoleHandler, instanceOfGuacamoleHandler } from './Guacamole';
import { useEffectAfterMount } from '../../services/helper';
import { UploadDialog } from './UploadDialog';
import { notify } from '../../App';

export default function Vnc() {

	const [ isLoading, setLoading ] = useState(true)
	const [ disconnectReason, setDisconnectReason ] = useState<{ code: "USER_ALREADY_EXISTS" | "UNKNOWN", message: string  } | null>(null)
	const [ settingsVisible, setSettingsVisible ] = useState(false)
	
	// Show a paste field for a short moment to be able to pase a text into the LFS.X 
	const [ showPasteField, setShowPasteField ] = useState(false)

	// Show an upload dialog to upload files to the LFS.X. The state contains the ID of the upload request
	// we need to send within the response of the uploaded files.
	// Because the UI thread of the LFS is halted, there can only be one request at a time
	const [ showUploadDialog, setShowUploadDialog ] = useState({ accept: "", id: 0 })

	// Coordinates of the current mouse position
	const mousePosition = useRef({ x: 0, y: 0 })

	const customizations = useCustomizations()

	// Select which client implementation should be used
	const [ useGuacamole, setUseGuacamole ] = React.useState(customizations.useGuacamole)

	const ref = useRef<VncScreenHandle|GuacamoleHandler>(null)

	const navigate = useNavigate()

	const [ toolbarItems, setToolbarItems ] = useState(getItems(navigate, () => setSettingsVisible(true)))

	// URL to connect to
	const baseURL = (location.protocol == "http:" ? "ws" : "wss") + "://" + location.host + "/api/vnc/ws"
	const url = baseURL + '?userIdentifier=' + encodeURIComponent(SecurityHelper.getUserIdentification())

	// Grab / ungrab keyboard for guacamole
	useEffect(() => {
		if (instanceOfGuacamoleHandler(ref.current)) {
			if (showPasteField) ref.current.ungrabKeyboard()
			else 				ref.current.grabKeyboard()
		}
	}, [ showPasteField ])
	useEffect(() => {
		if (instanceOfGuacamoleHandler(ref.current)) {
			if (showUploadDialog.id != 0) ref.current.ungrabKeyboard()
			else 				          ref.current.grabKeyboard()
		}
	}, [ showUploadDialog ])

	// Reconnect when display settings were changed
	useEffectAfterMount(() => {
		console.log("Reconecting because display settings were changed")
		// We don't want a reconnect from the client
		reconnectsOnClose.current = 10
		ref.current?.disconnect()
		setUseGuacamole(customizations.useGuacamole)

		setTimeout(() => {
			reconnectsOnClose.current = 10
			ref.current?.connect()
		}, 200)
		
	}, [ customizations.quality, customizations.useGuacamole ])
	// Apply settings
	useEffectAfterMount(() => {
		scaleWindowHot(customizations.scalingFactor).then((res) => {
			if (res) {
				console.log("Scaled window")
				notify("Die Schriftart und das Layout der Anwendung wird erst nach einem Neustart korrekt angewandt", 'info')
			} else  {
				console.log("Failed to scale window")
			} 
		}).catch((res: StandardResponse) => {
			console.error("Unable to scale window down: " + res.errorMessage)
		} )
	}, [ customizations.scalingFactor ])

	const reconnectsOnClose = useRef(0)
	const onSocketClose = (e: CloseEvent) => {
		console.log("Closed connection to WebSocket (" + e.code + ": " + e.reason + ")")

		// Determine the reason why the connection was closed
		probe(customizations).then(res => {
			setLoading(false)

			// The Websocket connection should work right awway
			if (res.status.code == 200) {
				setTimeout(() => {
					setLoading(true)
					ref.current?.connect()
				}, 1200)
				return
			} else {
				// Sometimes the VNC connection goes away (especially for guacamole: Error handling message from VNC server...).
				// In such case the probe was earlier than disconnecting cleanly -> retry to connect with a delay of 600 milliseconds
				if (reconnectsOnClose.current <= 1) {
					const millis = reconnectsOnClose.current == 0 ? 600 : 1200
					reconnectsOnClose.current = reconnectsOnClose.current + 1
					console.log("VNC server is gone away. Trying to reconnect in " + millis + " miliseconds")
					setTimeout(() => {
						setLoading(true)
						ref.current?.connect()
					}, millis)
				} else {
					console.log("Not trying to reconnect to VNC: reached retry limit")
					setDisconnectReason(getDisconnectReason(res))
				}

			}
		})
	}

	let lastResizeId = 0;
	const onResize = () => {

		// Clear previous timeouts
		clearTimeout(lastResizeId)

		// Schedule a new action to resize the windows after 100ms
		lastResizeId = setTimeout(() => {
			console.log("Sending reseize request")
			const size = getBrowserSize()
			resizeWindow(size.width, size.height)
		}, 100)
	}
	const getBrowserSize = () => {
		return {
			width: window.innerWidth
				|| document.documentElement.clientWidth
				|| document.body.clientWidth,
			height: window.innerHeight
				|| document.documentElement.clientHeight
				|| document.body.clientHeight
		}
	}

	let isCtrlDown = false
	/** Handles the pressing of a keyboard key */
	const onKeyType = (keysym: number, desc: string, down: boolean) => {

		// F11 -> go to fullscreen
		if (desc == "F11" && !down) {
			toogleFullscreen()
		}

		// Ctrl State is needed for some parts
		if (desc == "ControlLeft" || desc == "MetaLeft") {
			isCtrlDown = down

			// Reset paste field
			if (!down && showPasteField) {
				setShowPasteField(false)
				ref.current?.focus()
			}
		}

		// Paste into LFS.X
		if (isCtrlDown && desc === "KeyV" && customizations.clipBoardSupport) {
			console.log("Tried to paste")

			if (down) {
				setShowPasteField(!showPasteField)
			}
			
			return false
		}

		// console.log("Keyboard: " + keysym + " " + desc + " " + down)
		return desc != "F11"
	}

	/** Called when a WebSocket message was received from the LFS.X / controller */
	const onWebSocketMessage = (id: number, responseTo: number | null, message: WebSocketMessage) => {
		console.log("Received message from LFS.X WebSocket: " + message.type)

		if (message.type === "Stop") {
			console.log("Received a stop command from the LFS.X WebSocket")
			doLogout().then( success => success && navigate("/login"))
		} else if (message.type === "OpenInBrowser" && message.openInBrowser) {
			window.open(message.openInBrowser.url, '_blank')?.focus()
		} else if (message.type === "FileUploadRequest" && message.fileUploadRequest) {
			setShowUploadDialog({ accept: message.fileUploadRequest.accept, id: id })
		}
	}

	/** Sends a notification message to the LFS.X to state
	 * that the uploading of files was finished */
	const finishUpload = () => {
		send(showUploadDialog.id, {
			type: "FileUploadFinished",
		})
	}

	useEffect(() => {

		// If the api was already
		let fetchState = 0

		// Probe the connection for the first time
		setLoading(true)
		probe(customizations).then(res => {
			// Component got already cleaned up
			if (fetchState != 0) return

			// The Websocket connection should work -> connect
			if (res.status.code == 200) {
				fetchState = 1
				ref.current?.connect()
				connect(onWebSocketMessage)
			} else {
				setLoading(false)
				setDisconnectReason(getDisconnectReason(res))
			}
		})

		addEventListener("resize", onResize)

		// Cleanup function
		return () => {
			if (fetchState == 1 && ref.current != null) {
				console.log("Closing connection inside cleanup function")
				ref.current.disconnect()
			} else {
				fetchState = -1
			}

			removeEventListener("resize", onResize)
		}
	}, [])

	/**
	 * This function calls the "OnChange" callback for every menu items and updates them
	 * if they were changed
	 */
	const triggerUpdateForMenuitems = () => {
		const newItems = [...toolbarItems]
		let wasChanged = false

		for (let i = 0; i < newItems.length; i++) {
			if (newItems[i] && newItems[i].OnChange) {
				const newItem = newItems[i].OnChange?.(ref)
				if (newItem !== undefined && hasItemChanged(newItems[i], newItem)) {
					wasChanged = true
					newItems[i] = {...newItems[i], ...newItem }
				}
			}
		}

		// Only update state if one item was changed
		if (wasChanged) {
			setToolbarItems(newItems)
		}
	}

	return (
		<div>
			<div id="popup-container" onMouseEnter={triggerUpdateForMenuitems}>
				{ toolbarItems.filter( item => !item.Disabled && !(item.IsDisabled && item.IsDisabled(ref))).map( (item, i) => {
					return (
						<img 
							key={item.Icon} className='toolbar-item' 
							title={item.Tooltip} src={'/static/' + item.Icon} 
							onClick={() => item.OnClick(ref)}
							style={{  marginLeft: i === 0 ? "2px" : "7px" }}
						/>
					)
				}) }
			</div>

			<VncSettings setVisible={setSettingsVisible} visible={settingsVisible}/>

			{showPasteField && <div id="paste-field" style={{ top: mousePosition.current.y - 30, left: mousePosition.current.x - 50, zIndex: 10 }} > 
				<textarea 
					autoFocus={true} rows={4} cols={40} 
					onChange={ (e) => { 
						ref.current?.clipboardPaste(convertToLatin1(e.target.value));
						// Send STRG + V to lFS.X
						ref.current?.sendKey(65507, "ControlLeft", true)
						ref.current?.sendKey(118, "KeyV", true) 
						ref.current?.sendKey(118, "KeyV", false)
						ref.current?.sendKey(65507, "ControlLeft", false)
						onKeyType(0, "ControlLeft", false)
					}}
				/>
			</div>}

			<UploadDialog 
				visible={showUploadDialog.id != 0}
				setVisible={(visible: boolean) => {
					if (!visible) {
						console.log("Finishing uploading")
						finishUpload()
						setShowUploadDialog({ accept: "", id: 0 })
						
					}
				}}
				id={showUploadDialog.id}
				accept={showUploadDialog.accept}
			/>

			{ useGuacamole === false && <VncScreen
				className='vnc'
				url={url}
				scaleViewport={false}
				background="#e8e6e6"
				style={{
					width: "100%",
					height: "100%",
				}}
				ref={ref as any}
				// The quality levels doesn't seem to work correctly for wayvnc ...
				qualityLevel={customizations.quality === "low" ? 9 : customizations.quality == "medium" ? 5 : 8}		// Default: 6 | Max: 9 | More is better!
				compressionLevel={customizations.quality === "low" ? 2 : customizations.quality == "medium" ? 1 : 0}	// Default: 2 | Min: 0 (Disabled)
				onSocketCloseEvent={onSocketClose}
				onConnect={() => { onResize(); setDisconnectReason(null); reconnectsOnClose.current = 0 }}
				loadingUI={isLoading ? 
					<div className="loading-wrapper"> <LoadingAnimation text='Anwendung wird geladen' /></div> 
					: 
					<div> {disconnectReason?.message ?? "Unbekannter fehler"} </div>
				}
				autoConnect={false}
				retryDuration={5 * 1000}	// Only after 5 minutes
				onKeyType={onKeyType}
				onClipboard={(e) => e && customizations.clipBoardSupport && navigator.clipboard.writeText(convertFromLatin1(e?.detail.text)) }
				onMouseMove={(e) => mousePosition.current = { x: e.pageX, y: e.pageY }}
			/>}

			{ useGuacamole === true && <Guacamole 
				className='vnc'
				url={baseURL}
				ref={ref as any}
				onSocketClose={onSocketClose}
				disconnectReason={disconnectReason}
				onConnect={() => { onResize(); setDisconnectReason(null); reconnectsOnClose.current = 0 }}
				onKeyType={onKeyType}
				onMouseMove={(e) => mousePosition.current = { x: e.x, y: e.y }}
			/>}

		</div>
	);

}

/**
	* This function determines the reason why the connection to the WebSocket failed. Afterwards the 
	* state will be set that the client does see the error
	* 
	* @param res 	The response of the probe action
*/
export function getDisconnectReason(res: StandardResponse): { code: "USER_ALREADY_EXISTS" | "UNKNOWN", message: string } {
	if (res.data === null) {
		return{code: "UNKNOWN", message: "Es trat ein unbekannter Fehler auf"}
	} else {
		const message = res.data.message == null ? res.data : res.data.message
	
		switch (message) {
			case "USER_ALREADY_EXISTS": {
				return{code: "USER_ALREADY_EXISTS", message: "Die bist bereits in einem anderen Fenster mit der Anwendung verbunden"}
				break;
			}
			default: {
				console.log("Unknown disconnect reason: " + message)
				return {code: "UNKNOWN", message: "Es trat ein unbekannter Fehler auf"}
			}
		}
	}
}

/**
 * Converts the given string to a UTF-8 JavaScript string.
 * 
 * @param str 	Latin-1 (ISO 8859-1) String to convert
 */
export function convertFromLatin1(str: string): string {
	return str
		.replaceAll("\u00c3\u00a4", "ä")
		.replaceAll("\u00c3\u0084", "Ä")
		.replaceAll("\u00c3\u00bc", "ü")
		.replaceAll("\u00c3\u009c", "Ü")
		.replaceAll("\u00c3\u0096", "Ö")
		.replaceAll("\u00c3\u00b6", "ö")
		.replaceAll("\u00c3\u009f", "ß")
}

/**
 * Converts the given string to the ISO 8859-1 charset that VNC 
 * does understand for clipboard support
 * 
 * @param str 	UTF-8 String to convert
 */
export function convertToLatin1(str: string): string {
	return str
		.replaceAll("ä", "\u00c3\u00a4")
		.replaceAll("Ä", "\u00c3\u0084")
		.replaceAll("ü", "\u00c3\u00bc")
		.replaceAll("Ü", "\u00c3\u009c")
		.replaceAll("Ö", "\u00c3\u0096")
		.replaceAll("ö", "\u00c3\u00b6")
		.replaceAll("ß", "\u00c3\u009f")
		//return str.replaceAll("ä", "\u00e4")
}