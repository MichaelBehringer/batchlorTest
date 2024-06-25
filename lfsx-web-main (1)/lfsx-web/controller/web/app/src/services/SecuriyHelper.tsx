import { configureAxios } from "./RequestService";

class SecurityHelper {

	private static cookieName = "JWTAuthentication"
	private static cookiePath = "/"
	private static userIdentifier = ""

	public static clearAllLoginCredentials = () => {
		const domain = window.location.hostname

		if (SecurityHelper.getCookie(this.cookieName)) {
			document.cookie = this.cookieName + "=" +
                ((this.cookiePath) ? ";path=" + this.cookiePath : "") +
                ((domain) ? ";domain=" + domain : "") +
                ";expires=Thu, 01 Jan 1970 00:00:01 GMT";
		}
		if (localStorage.getItem('isLoggedIn') === "true") localStorage.setItem('isLoggedIn', 'false')
		configureAxios()
	}

	private static getCookie = (name: string) => {
		return document.cookie.split(';').some(c => {
			return c.trim().startsWith(name + '=');
		});
	}

	public static isAuthorized(): boolean {
		return localStorage.getItem('isLoggedIn') === "true"
	}
	public static setAuthoirzed() {
		localStorage.setItem('isLoggedIn', "true")
	}

	public static redirectToLogin = () => {
		window.location.href =  location.protocol + '//' + location.host + "/login";
	}

	public static setUserIdentifier(identifier: string) {
		this.userIdentifier = identifier
	}

	public static getUserIdentification(): string {
		if (this.userIdentifier === "") return Math.random().toString()
		else 							return this.userIdentifier
	}
}

export default SecurityHelper