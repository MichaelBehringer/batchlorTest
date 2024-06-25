import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'
import CustomizationProvider from './provider/CustomizationProvider'


ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
	<React.StrictMode>
		<CustomizationProvider>
			<App />
		</CustomizationProvider>
	</React.StrictMode>,
)
