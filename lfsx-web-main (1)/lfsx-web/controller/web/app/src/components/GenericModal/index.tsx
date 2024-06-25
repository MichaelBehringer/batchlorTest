import { useEffect, useState } from 'react'
import './index.css'

/**
 * GenericModal is a modal tha
 */
export function GenericModal(props: GenericModalProps) {

	const [delayedVisible, setDelayedVisible] = useState(props.visible)

	useEffect(() => {
		setTimeout(() => {
			setDelayedVisible(props.visible)
		}, props.visible ? 0 : 450)
	}, [props.visible])


	return (
		<div className='modal-root-wrapper' data-visible={delayedVisible ? "true" : "false"} >
			<div className='modal-dark-mask' data-visible={props.visible ? "true" : "false"} />

			<div className='modal-wrapper' data-visible={props.visible ? "true" : "false"}>
				<div className='modal-header' data-visible={props.visible ? "true" : "false"} >
					<span className='close' onClick={() => props.setVisible(false)}>&times;</span>
					<h3>{props.title}</h3>
				</div>

				<div className='modal-body' >
					{props.children}
				</div>
			</div>
		</div>
	)
}

export type GenericModalProps = {

	/** Update the visible state */
	setVisible: (visible: boolean) => void

	/** If the modal is visible */
	visible: boolean

	/** Title of the modal */
	title: string

	/** Component to display inside the modal  */
	children: React.ReactNode
}