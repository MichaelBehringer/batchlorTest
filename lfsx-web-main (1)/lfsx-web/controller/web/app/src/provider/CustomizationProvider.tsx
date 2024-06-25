import React, { createContext, PropsWithChildren, useContext, useEffect, useState } from 'react'
import { Login } from '../data/login'
import SecurityHelper from '../services/SecuriyHelper'

const isBrowserDefaultDark = () => window.matchMedia('(prefers-color-scheme: dark)').matches

const CONFIG = "appConfig"

export const CustomizationContext = createContext<CustomizationProviderInterface | null>(null)

const CustomizationProvider: React.FunctionComponent<PropsWithChildren> = (props) => {
	const [dataProvider, setDataProvider] = useState<CustomizationValues>(init())

	function handleUpdate(val: CustomizationValues) {
		localStorage.setItem(CONFIG, JSON.stringify(val))
		setDataProvider(val)
	}

	useEffect(() => {
		if (dataProvider.login) SecurityHelper.setUserIdentifier(dataProvider.login.user + "-" + dataProvider.login.db)
	}, [ dataProvider.login ])

	function init(): CustomizationValues {
		const strConfig = localStorage.getItem("appConfig")
		if (strConfig !== null) {
			const values = JSON.parse(strConfig) as CustomizationValues

			// Set initial value for security helper
			if (values.login) SecurityHelper.setUserIdentifier(values.login.user + "-" + values.login.db)
			if (values.useGuacamole === undefined) values.useGuacamole = false

			return values
		} else {
			return {
				clipBoardSupport: true,
				darkMode: isBrowserDefaultDark(),
				quality: "high",
				useGuacamole: false,
				scalingFactor: 100
			}
		}
	}

	return (<CustomizationContext.Provider value={{ dataProvider, setDataProvider: handleUpdate }}>
		{props.children}
	</CustomizationContext.Provider>)
}

export function useCustomizations(): CustomizationValues {
	const dataProvider = useContext(CustomizationContext)
	
	// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
	return dataProvider!.dataProvider
}

export default CustomizationProvider

export interface CustomizationProviderInterface {
    dataProvider: CustomizationValues,
    setDataProvider: ( (provider: CustomizationValues) => void )
}

export interface CustomizationValues {
	clipBoardSupport: boolean
	quality: "low" | "medium" | "high"
	darkMode: boolean
	login?: Login
	useGuacamole: boolean
	scalingFactor: 100 | 125 | 150 | 175 | 200
}