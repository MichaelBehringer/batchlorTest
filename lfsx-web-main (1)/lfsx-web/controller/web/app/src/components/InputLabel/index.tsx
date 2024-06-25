import React from 'react'
import './index.css'

export default function InputLabel(props: InputProps) {
	return (
		<div className="input-wrapper">
			{props.image && <img src={'/static/' + props.image} className='label-icon' />}
			<input className="input-label-input" name={props.name} type={props.type} value={props.value} onChange={props.onChange} placeholder={props.placeholder} />
		</div>
	)
}

export interface InputProps {
	/* Name of the input element for usage inside a form */
	name: string;

	/* Type of the input element like "text" or "password" */
	type?: string;

	/* Relative image path to display in front of the text input (inside folder 'static') */
	image?: string;
	placeholder?: string;

	/* The value of the input element */
	value: string | number | readonly string[] | undefined;
	/* Function to call when the user does change the value */
	onChange: React.ChangeEventHandler<HTMLInputElement> | undefined; 
}