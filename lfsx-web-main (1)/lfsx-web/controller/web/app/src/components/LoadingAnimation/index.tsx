import React from 'react'
import './index.css'

export default function LoadingAnimation(props: LoadingAnimationProps) {
	return (
		<div className="spinner-box loading">
			<div className="blue-orbit leo">
			</div>

			<div className="green-orbit leo">
			</div>
  
			<div className="red-orbit leo">
			</div>
  
			<div className="white-orbit w1 leo">
			</div><div className="white-orbit w2 leo">
			</div><div className="white-orbit w3 leo">
			</div>

			<div className='text'>{props.text}</div>
		</div>
	)

}

export interface LoadingAnimationProps {
	text?: string;
}