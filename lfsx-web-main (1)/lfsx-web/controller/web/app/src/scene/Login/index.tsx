import React, {FormEvent, useContext, useState} from 'react'
import 'react-toastify/dist/ReactToastify.css';
import { useNavigate } from 'react-router-dom'
import { notify } from '../../App'
import { doLogin, Login } from '../../data/login'
import './index.css'
import InputLabel from '../../components/InputLabel';
import { CustomizationContext } from '../../provider/CustomizationProvider';
import { VncSettings } from '../Vnc/Settings';


export default function LoginScene() {

	const dataProvider = useContext(CustomizationContext)
	const navigate = useNavigate()
	const [ settingsVisible, setSettingsVisible ] = useState(false)

	// A list of available databases for the app
	let dbs = [ "lfs" ]
	if (Config.prod === false) {
		dbs = [ "lfsmig", "lfsprj", "lfs" ]
	}
    
	const emptyLogin = {login: "", password: "", db: dbs[0], user: ""}
	const [ values, setValues ] = useState<Login>(emptyLogin)

	return (
		<>
			<VncSettings
				setVisible={setSettingsVisible}
				visible={settingsVisible}
				key={"login-settings"}
			/>

			<div id="login-mask">

				<img 
					className='settings' 
					title={"Settings"} src='/static/settings-black.svg'
					onClick={() => { setSettingsVisible(true) }}
				/>

				<h2> Anmeldung </h2>

				<form onSubmit={handleLogin} className='wrapper'>

					<InputLabel 
						type="text" name="login" 
						value={values.login} onChange={ (ev) => setValues({...values, login: ev.target.value}) } 
						image='user.svg' placeholder='Benutzername'
					/>
					<InputLabel 
						type="password" name="password" 
						value={values.password} onChange={ (ev) => setValues({...values, password: ev.target.value}) } 
						image='lock.svg' placeholder='Passwort'
					/>

					<label>
						<select value={values.db} name="db" onChange={ (ev) => setValues({...values, db: ev.target.value}) }>
							{dbs.map(db => 
								<option value={db} key={db}>{db}</option>    
							)}
						</select>
					</label>

					<input type="submit" value="Anmelden" />
				</form>
			</div>
		</>
	)

	function handleLogin(event: FormEvent) {
		event.preventDefault()

		if (values.login == "" || values.password == "") {
			notify("Benutzername und Password sind erforderlich", "error")
			return
		}

		doLogin(values).then(rtc => {
			if (rtc.message == "") {
				localStorage.setItem("isLoggedIn", "true")
				dataProvider?.setDataProvider({...dataProvider.dataProvider, login: rtc.data})
				navigate("/")
				setValues(emptyLogin)
			} else {
				notify(rtc.message, "error")
			}
		}).catch( () => notify("Unbekannter Fehler", "error"))
	}
}

