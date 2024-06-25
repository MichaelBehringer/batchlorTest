import { NavigateFunction } from "react-router-dom"
import { VncScreenHandle } from "../../components/VncScreen"
import { doLogout } from "../../data/login"
import { GuacamoleHandler } from "./Guacamole"

/**
 * Toolbar presents an item on the toolbar to performa various actions via a popup container.
 */
export interface ToolbarItem {

	/* Icon to display as a button on the toolbar */
	Icon: string

	/* Text to show when hovering over an item */
	Tooltip: string
	
	/* Unique key assigned to this item to perform rerenders efficently */
	Key: string

	/* Action to perform when the toolbar entry was clicked */
	OnClick: (vncScreen: React.MutableRefObject<VncScreenHandle|GuacamoleHandler|null>) => void

	/* Function that is called when something was changed. You can use that to change the internal item properties.  */
	OnChange?: (vncScreen: React.MutableRefObject<VncScreenHandle|GuacamoleHandler|null>) => { Icon: string, Tooltip: string, Disabled?: boolean } | null

	/* Function to check if the menu entry should be disabled */
	IsDisabled?: (vncScreen: React.MutableRefObject<VncScreenHandle|GuacamoleHandler|null>) => boolean

	/* If this item is currently disabled */
	Disabled?: boolean
}

/**
 * This function returns a list of initial items that should be displayed on the toolbar.
 * 
 * @returns 	List of items
 */
export function getItems(navigate: NavigateFunction, showSettingsModal: () => void): ToolbarItem[] {
	return [
		{
			Icon: "fullscreen.svg", Tooltip: "Vollbildmodus umschalten", Key: "fullscren",
			OnClick: () => { toogleFullscreen() },
			OnChange: () => {
				if (!isFullScreenEnabled()) {
					return { Icon: "fullscreen.svg", Tooltip: "In Vollbildmodus wechseln" }
				} else {
					return { Icon: "fullscreen-exit.svg", Tooltip: "Vollbildmodus verlassen" }
				}
			}
		},
		{
			Icon: "logout.svg", Tooltip: "Abmelden", Key: "logout",
			OnClick: () => {
				doLogout().then( success => success && navigate("/login"))
			}
		},
		{
			Icon: "settings.svg", Tooltip: "Einstellungen", Key: "settings",
			OnClick: showSettingsModal
		},
	]
}

/**
 * Switches the fullscreen mode from none -> fullscreen or from fullscreen -> none
 * based on the current fullscren status.
 * 
 * Note that this function can only be called from a user thread that has been running
 * for at least 1 seconds. Otherwise the switch from none -> fullscreen will be rejected.
 */
export function toogleFullscreen() {
	const elem = document.documentElement;
	const isFullscreen = isFullScreenEnabled()

	if (!isFullscreen && elem.requestFullscreen) {
		elem.requestFullscreen();
	} else if (isFullscreen && document.exitFullscreen) {
		document.exitFullscreen()
	}
}
const isFullScreenEnabled = () => (!window.screenTop && !window.screenY) || document.fullscreenElement

/**
 * Returns weather the old item isn't equal to the new item
 * 
 * @param oldItem 	Item to compare
 * @param newItem 	Item to compare
 * 
 * @returns			If the item was changed 
 */
export function hasItemChanged(oldItem: ToolbarItem, newItem: { Icon: string, Tooltip: string, Disabled?: boolean } | null) {
	if (oldItem == null || newItem == null) return false

	return oldItem.Icon !== newItem.Icon || oldItem.Disabled !== newItem.Disabled || oldItem.Tooltip !== newItem.Tooltip
}