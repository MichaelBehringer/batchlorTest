import { useContext, useState } from 'react'
import { CustomizationContext, useCustomizations } from '../../../provider/CustomizationProvider'
import './index.css'
import { useEffectAfterMount } from '../../../services/helper'
import { GenericModal } from '../../../components/GenericModal'

export function VncSettings(props: VncSettingsProps) {

	const dataProvider = useContext(CustomizationContext)
	const cust = useCustomizations()

	const [ values, setValues ] = useState(cust)

	// Write the locally stored values (inside this component) to the customization provider
	useEffectAfterMount(() => {
		console.log("Persisting settings")
		dataProvider?.setDataProvider(values)
	}, [ props.visible ])


	return (
		<GenericModal
			visible={props.visible}
			setVisible={props.setVisible}
			title='Einstellungen'
		>
			<label>
				<input type="checkbox" checked={values.clipBoardSupport} onChange={() => {
					setValues({...values, clipBoardSupport: !values.clipBoardSupport})
				}}/>
						Aktiviere Copy-und-Paste Funktion zwischen LFS.X und Client
			</label>
			<br />

			<label className="grid">
						Bildqualität:
				<select value={values.quality} name="quality" id="quality" 
					onChange={ (ev) => setValues({...values, quality: ev.target.value as "low" | "medium" | "high"}) }>
					{[ 
						{ label: "Niedrig", value: "low" }, 
						{ label: "Mittel", value: "medium" },
						{ label: "Hoch", value: "high"}
					].map(qual => 
						<option value={qual.value} key={qual.value}>{qual.label}</option>    
					)}
				</select>
			</label>
			<br />

			<label>
				<input type="checkbox" checked={values.useGuacamole} onChange={() => {
					setValues({...values, useGuacamole: !values.useGuacamole})
				}}/>
						Verwende das Guacamole Protokoll anstelle von VNC
			</label>
			<br />

			<label className="grid">
					Skalierung:
				<select value={values.scalingFactor} name="scaling" id="scaling" 
					onChange={ (ev) => setValues({...values, scalingFactor: Number(ev.target.value) as 100 | 125 | 150 | 175 | 200}) }>
					{[ 
						{ label: "100%", value: 100 },
						{ label: "125%", value: 125 },
						{ label: "150%", value: 150 },
						{ label: "175%", value: 175 },
						{ label: "200%", value: 200 }
					].map(qual => 
						<option value={qual.value} key={qual.value}>{qual.label}</option>    
					)}
				</select>
			</label>
			<br />
				
		</GenericModal>
	)
}

export type VncSettingsProps = {
	setVisible: (visible: boolean) => void
	visible: boolean
}