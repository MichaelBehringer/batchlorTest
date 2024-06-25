import React, { useEffect } from 'react';
import {BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate} from 'react-router-dom';
import { notify } from '../../App';
import Login from '../../scene/Login';
import Vnc from '../../scene/Vnc';
import getRequest from '../../services/RequestService';


const PrivateRoute = ({children}: {children: JSX.Element}): JSX.Element => {
	const location = useLocation();
	return localStorage.getItem("isLoggedIn") === "true" ? children : <Navigate to="/login" state={{from: location}} />;
}

const PrivateRouteUnauthorized = ({children}: {children: JSX.Element}): JSX.Element => {
	const location = useLocation();
	return localStorage.getItem("isLoggedIn") === "true" ? <Navigate to="/" state={{from: location}} /> : children;
}

export default function AppRoutes() {

	useEffect(() => {

		// Check if user is still logged in on first startup
		if (localStorage.getItem("isLoggedIn") === "true") {
			getRequest("/isAuthenticated", null, null, {}, { method: "get" })
				.then(res => {
					if (res.status.code == 401 || res.status.code == 403) {
						localStorage.setItem("isLoggedIn", "false")
						window.location.href = "/login"
					} else if (res.status.code == 200) {
					// The user is stille logged in
					} else {
						notify("Keine Verbindung", "warning")
					}
				}).catch(() => notify("Keine Verbindung", "warning"))
		}

	}, [])

	return (
		<BrowserRouter>
			<Routes>
            
				{/* Login */}
				<Route path="/login" element={
					<PrivateRouteUnauthorized>
						<Login />
					</PrivateRouteUnauthorized>
				} />

				<Route path="/*" element={
					<PrivateRoute>
						<Routes>
							<Route path="/vnc" element={< Vnc/>} />

							<Route path="*" element={<Navigate to="/vnc" />} />
						</Routes>
					</PrivateRoute>
				}>
				</Route>
			</Routes>
		</BrowserRouter>
	)
}
