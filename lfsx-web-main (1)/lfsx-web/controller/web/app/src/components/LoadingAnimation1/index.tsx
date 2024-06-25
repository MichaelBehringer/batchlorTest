import React from 'react'
import './index.css'

export default function LoadingAnimation1(props: LoadingAnimationProps) {
	return (
		<div className="loader">
			<div className="followers_arc_outer followers_arc_start o_circle"></div>
			<div className="followers_arc_inner followers_arc_start i_circle"> </div>
			<div className="text">{props.text}</div>
		</div>
	)

}

export interface LoadingAnimationProps {
	text?: string;
}