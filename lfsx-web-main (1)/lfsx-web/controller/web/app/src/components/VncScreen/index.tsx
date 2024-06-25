import React, { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
import RFB from '../NoVNC/core/rfb';

/**
 * This code is based on https://github.com/roerohan/react-vnc/tree/main/src/lib by roerohan with
 * small changes to make it compatible with React 18.
 * 
 * RFB Docs of noVNC are available here: https://novnc.com/noVNC/docs/API.html
 */

export interface RFBOptions {
    shared: boolean;
    credentials: {
        username?: string;
        password?: string;
        target?: string;
    };
    repeaterID: string;
    wsProtocols: string;
}

export interface Props {
    url: string;
    style?: object;
    className?: string;
    viewOnly?: boolean;
    rfbOptions?: Partial<RFBOptions>;
    focusOnClick?: boolean;
    clipViewport?: boolean;
    dragViewport?: boolean;
    scaleViewport?: boolean;
    resizeSession?: boolean;
    showDotCursor?: boolean;
    background?: string;
    qualityLevel?: number;
    compressionLevel?: number;
    autoConnect?: boolean;
    retryDuration?: number;
    debug?: boolean;
    loadingUI?: React.ReactNode;
    onConnect?: (rfb?: RFB) => void;
    onDisconnect?: (rfb?: RFB) => void;
	onSocketCloseEvent?: (event: CloseEvent) => void;
    onCredentialsRequired?: (rfb?: RFB) => void;
    onSecurityFailure?: (e?: { detail: { status: number, reason: string } }) => void;
    onClipboard?: (e?: { detail: { text: string } }) => void;
    onBell?: () => void;
    onDesktopName?: (e?: { detail: { name: string } }) => void;
    onCapabilities?: (e?: { detail: { capabilities: RFB["capabilities"] } }) => void;

	/* This event is fired when a key down / key up event was registred. The return value indicates if the key event 
	   should be passed to the VNC instance	*/
	onKeyType?: (symbol: number, descr: string, down: boolean) => boolean;

	/* This event is fired when the mouse is moving inside the VNC screen */
	onMouseMove?: (ev: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
}

export enum Events {
    connect,
    disconnect,
    credentialsrequired,
    securityfailure,
    clipboard,
    bell,
    desktopname,
    capabilities,
}

export type EventListeners = { -readonly [key in keyof typeof Events]?: (e?: any) => void };

export type VncScreenHandle = {
    connect: () => void;
    disconnect: () => void;
    connected: boolean;
    sendCredentials: (credentials: RFBOptions["credentials"]) => void;
    sendKey: (keysym: number, code: string, down?: boolean) => void;
    sendCtrlAltDel: () => void;
    focus: () => void;
    blur: () => void;
    machineShutdown: () => void;
    machineReboot: () => void;
    machineReset: () => void;
    clipboardPaste: (text: string) => void;
    rfb: RFB | null;
    eventListeners: EventListeners;
};

const VncScreen: React.ForwardRefRenderFunction<VncScreenHandle, Props> = (props, ref) => {
	const rfb = useRef<RFB | null>(null);
	const connected = useRef<boolean>(props.autoConnect ?? true);
	const timeouts = useRef<Array<ReturnType<typeof setTimeout>>>([]);
	const eventListeners = useRef<EventListeners>({});
	const screen = useRef<HTMLDivElement>(null);
	const [loading, setLoading] = useState<boolean>(true);

	const {
		url,
		style,
		className,
		viewOnly,
		rfbOptions,
		focusOnClick,
		clipViewport,
		dragViewport,
		scaleViewport,
		resizeSession,
		showDotCursor,
		background,
		qualityLevel,
		compressionLevel,
		autoConnect = true,
		retryDuration = 3000,
		debug = false,
		loadingUI,
		onConnect,
		onDisconnect,
		onSocketCloseEvent,
		onCredentialsRequired,
		onSecurityFailure,
		onClipboard,
		onBell,
		onDesktopName,
		onCapabilities,
	} = props;

	const logger = {
		log: (...args: any[]) => { if (debug) console.log(...args); },
		info: (...args: any[]) => { if (debug) console.info(...args); },
		error: (...args: any[]) => { if (debug) console.error(...args); },
	};

	const getRfb = () => {
		return rfb.current;
	};

	const setRfb = (_rfb: RFB | null) => {
		rfb.current = _rfb;
	};

	const getConnected = () => {
		return connected.current;
	};

	const setConnected = (state: boolean) => {
		connected.current = state;
	};

	const _onConnect = () => {
		const rfb = getRfb();
		if (onConnect) {
			onConnect(rfb ?? undefined);
			setLoading(false);
			return;
		}

		logger.info('Connected to remote VNC.');
		setLoading(false);
	};

	const _onDisconnect = () => {
		const rfb = getRfb();
		if (onDisconnect) {
			onDisconnect(rfb ?? undefined);
			setLoading(true);
			return;
		}

		const connected = getConnected();
		if (connected) {
			logger.info(`Unexpectedly disconnected from remote VNC, retrying in ${retryDuration / 1000} seconds.`);

			timeouts.current.push(setTimeout(connect, retryDuration));
		} else {
			logger.info(`Disconnected from remote VNC.`);
		}
		setLoading(true);
	};

	const _onCredentialsRequired = () => {
		const rfb = getRfb();
		if (onCredentialsRequired) {
			onCredentialsRequired(rfb ?? undefined);
			return;
		}

		const password = rfbOptions?.credentials?.password ?? prompt("Password Required:");
		rfb?.sendCredentials({ password: password });
	};

	const _onDesktopName = (e: { detail: { name: string } }) => {
		if (onDesktopName) {
			onDesktopName(e);
			return;
		}

		logger.info(`Desktop name is ${e.detail.name}`);
	};

	const disconnect = () => {
		const rfb = getRfb();
		try {
			if (!rfb) {
				return;
			}

			timeouts.current.forEach(clearTimeout);
			(Object.keys(eventListeners.current) as (keyof typeof Events)[]).forEach((event) => {
				if (eventListeners.current[event]) {
					rfb.removeEventListener(event, eventListeners.current[event]);
					eventListeners.current[event] = undefined;
				}
			});
			rfb.disconnect();
			setRfb(null);
			setConnected(false);

			// NOTE(roerohan): This needs to be called since the event listener is removed.
			// Even if the event listener is removed after rfb.disconnect(), the disconnect
			// event is not fired.
			_onDisconnect();
		} catch (err) {
			logger.error(err);
			setRfb(null);
			setConnected(false);
		}
	};

	const connect = () => {
		try {
			if (connected && !!rfb) {
				disconnect();
			}

			if (!screen.current) {
				return;
			}

			screen.current.innerHTML = '';

			const _rfb = new RFB(screen.current, url, rfbOptions);

			_rfb.viewOnly = viewOnly ?? false;
			_rfb.focusOnClick = focusOnClick ?? false;
			_rfb.clipViewport = clipViewport ?? false;
			_rfb.dragViewport = dragViewport ?? false;
			_rfb.resizeSession = resizeSession ?? false;
			_rfb.scaleViewport = scaleViewport ?? false;
			_rfb.showDotCursor = showDotCursor ?? false;
			_rfb.background = background ?? '';
			_rfb.qualityLevel = qualityLevel ?? 6;
			_rfb.compressionLevel = compressionLevel ?? 2;
			_rfb._sock.on('close', (e: CloseEvent) => {
				_rfb._socketClose(e)
				if (onSocketCloseEvent != null) {
					onSocketCloseEvent(e)
				}
			})

			// When VNC is binding on key events the generic javascript event (keydown) won't be called.
			// => overwrite this function and pass call through
			_rfb._keyboard.onkeyevent = (keysym: number, code: string, down: boolean) => {
				if (props.onKeyType) {
					if (props.onKeyType(keysym, code, down)) {
						_rfb._handleKeyEvent(keysym, code, down)
					}
				} else {
					_rfb._handleKeyEvent(keysym, code, down)
				}
			}
			// VNC does also overide the mouse move event
			_rfb._canvas.addEventListener('mousemove', (ev: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
				if (props.onMouseMove) {
					props.onMouseMove(ev)
				}
			})

			setRfb(_rfb);

			eventListeners.current.connect = _onConnect;
			eventListeners.current.disconnect = _onDisconnect;
			eventListeners.current.credentialsrequired = _onCredentialsRequired;
			eventListeners.current.securityfailure = onSecurityFailure;
			eventListeners.current.clipboard = onClipboard;
			eventListeners.current.bell = onBell;
			eventListeners.current.desktopname = _onDesktopName;
			eventListeners.current.capabilities = onCapabilities;

			(Object.keys(eventListeners.current) as (keyof typeof Events)[]).forEach((event) => {
				if (eventListeners.current[event]) {
					_rfb.addEventListener(event, eventListeners.current[event]);
				}
			});

			setConnected(true);
		} catch (err) {
			logger.error(err);
		}
	};

	const sendCredentials = (credentials: RFBOptions["credentials"]) => {
		const rfb = getRfb();
		rfb?.sendCredentials(credentials);
	};

	const sendKey = (keysym: number, code: string, down?: boolean) => {
		const rfb = getRfb();
		rfb?.sendKey(keysym, code, down);
	};

	const sendCtrlAltDel = () => {
		const rfb = getRfb();
		rfb?.sendCtrlAltDel();
	};

	const focus = () => {
		const rfb = getRfb();
		rfb?.focus();
	};

	const blur = () => {
		const rfb = getRfb();
		rfb?.blur();
	};

	const machineShutdown = () => {
		const rfb = getRfb();
		rfb?.machineShutdown();
	};

	const machineReboot = () => {
		const rfb = getRfb();
		rfb?.machineReboot();
	};

	const machineReset = () => {
		const rfb = getRfb();
		rfb?.machineReset();
	};

	const clipboardPaste = (text: string) => {
		const rfb = getRfb();
		rfb?.clipboardPasteFrom(text);
	};

	useImperativeHandle(ref, () => ({
		connect,
		disconnect,
		connected: connected.current,
		sendCredentials,
		sendKey,
		sendCtrlAltDel,
		focus,
		blur,
		machineShutdown,
		machineReboot,
		machineReset,
		clipboardPaste,
		rfb: rfb.current,
		eventListeners: eventListeners.current,
	}));

	useEffect(() => {
		if (autoConnect) {
			connect();
		}

		return disconnect;
	}, []);

	const handleClick = () => {
		const rfb = getRfb();
		if (!rfb) return;

		rfb.focus();
	};

	const handleMouseEnter = () => {
		if (document.activeElement && document.activeElement instanceof HTMLElement) {
			document.activeElement.blur();
		}

		handleClick();
	};

	const handleMouseLeave = () => {
		const rfb = getRfb();
		if (!rfb) {
			return;
		}

		rfb.blur();
	};

	return (
		<>
			<div
				style={style}
				className={className}
				ref={screen}
				onMouseEnter={handleMouseEnter}
				onMouseLeave={handleMouseLeave}
			/>
			{loading && (loadingUI ?? <div className="text-white loading">Loading...</div>)}
		</>
	);
}

export default forwardRef(VncScreen);
