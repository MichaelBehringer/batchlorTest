import { CustomizationValues } from "../provider/CustomizationProvider"
import { RequestHelper, StandardResponse } from "../services/RequestService"

export async function resizeWindow(width: number, height: number): Promise<boolean> {
	return RequestHelper.post("/host/vnc/resolution", {width: width, height: height}).then((res) => {
		return res.status.code === 200
	})
}

export async function scaleWindowHot(scalingFactor: number): Promise<boolean> {
	return RequestHelper.post("/host/vnc/scale", {factor: scalingFactor}).then((res) => {
		return res.status.code === 200
	})
}

export async function probe(settings: CustomizationValues): Promise<StandardResponse> {
	return RequestHelper.get("/vnc/ws/probe", {
		"scale": settings.scalingFactor
	})
}

export type VncSettings = {
	Scaling: number
}