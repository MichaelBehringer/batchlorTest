import './App.css'
import AppRoutes from './components/Routing'
import { ToastContainer, toast, TypeOptions } from 'react-toastify';

function App() {
	
	return (
		<>
			<ToastContainer limit={3} />
			<AppRoutes />
		</>
	)
}

export function notify(message: string, type: TypeOptions, theme?: "light" | "dark" | "default") {
	if (!theme || theme === "default") {
		theme = document.getElementById("dark") ? "dark" : "light"
	}
  
	toast(message, {
		position: "top-right",
		autoClose: 4000,
		hideProgressBar: false,
		closeOnClick: true,
		pauseOnHover: true,
		draggable: true,
		progress: undefined,
		theme: theme ?? "light",
		type: type
	})
}

export default App
