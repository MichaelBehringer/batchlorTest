import getRequest from "../services/RequestService"

export async function doLogin(body: Login): Promise<{ data?: Login, message: string }> {

	// Convert json to form data
	const formData = new URLSearchParams();
	Object.keys(body).forEach( key => formData.append(key, body[key as keyof Login]));
	formData.append("origin", "LFS")

	return getRequest(
		"/login",
		null, 
		formData.toString(),
		{ 'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8' },
		{ acceptCookieSetHeader: true, method: "post" }
	).then((res) => {
		if (res.status.code == 401 || res.status.code == 403) {
			return { message: "Benutzername oder Passwort sind ungültig" }
		} else if (res.status.code == 200) {
			// Login was successfull
			console.log(res.data)
			return { message: "", data: res.data }
		} else {
			return { message: "Unbekannter Fehler" }
		}
	})
}

export async function doLogout(): Promise<boolean> {
	return getRequest(
		"/logout",
		null, null, {},
		{ acceptCookieSetHeader: true, method: "post" }
	).then((res) => {
		return res.status.code === 200
	})
}

export type Login = {
	login: string
	password: string
	db: string
	user: string
}