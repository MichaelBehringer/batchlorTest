import './index.css'
import { GenericModal } from '../../../components/GenericModal'
import { useEffect, useRef, useState } from 'react';
import getRequest from '../../../services/RequestService';
import { notify } from '../../../App';

export function UploadDialog(props: UploadDialogProps) {

	const [ files, setFiles ] = useState<CustomFile[]>([])
	/* Number of files that are curently be uploaded */
	const currentUploads = useRef(0);
	const [ isDragging, setDragging ] = useState(false)
	const fileInputRef = useRef<HTMLElement>();

	// Clear all files on open
	useEffect(() => {
		if (props.visible) setFiles([])
	}, [props.visible])

	function uploadFile(file: CustomFile) {
		// Add to file list
		file.progress = 0
		currentUploads.current = currentUploads.current +1
		setFiles((prevFiles ) => { return [...prevFiles, file]  })

		getRequest("/app/file/" + props.id + "/upload", null, file.file, { "Filename": file.name }, {
			onUploadProgress: (progress) => {
				setFiles(( prevFiles => {
					return [...prevFiles].map(f => {
						// Update the file
						if (f.id === file.id && progress.progress && f.progress !== 100) {
							let newProgress = Math.round(progress.progress * 100)
							// Don't allow to set a progress > 100. This is only set if the request was successfully finished!
							if (newProgress >= 100) {
								newProgress = 99
							}
							console.log(f.progress + " -> " + newProgress)
							f.progress = newProgress
						}
						return f
					})
				}))
			},
			method: "post"
		}).then(
			(res) => {
				if (res.status.code === 200) {
					console.log("Finished to upload file: " + file.name)
					setFiles(( prevFiles => {
						return [...prevFiles].map(f => {
							// Update the file
							if (f.id === file.id) {
								f.progress = 100
							}
							return f
						})
					}))
				}
			})
			.finally(() => {
				currentUploads.current = currentUploads.current - 1
			})
	}

	useEffect(() => {
		const input = document.createElement('input')
		input.type = 'file'
		input.accept = props.accept
		input.multiple = true

		input.onchange = () => {
			if (input.files && input.files.length > 0) {
				const droppedFiles = Array.from(input.files).map(i => { return {progress: 0, name: i.name, extension: getFileExtension(i.name), size: i.size, file: i, id: (Math.random() + 1).toString(36).substring(2) }  })
				droppedFiles.forEach(f => {
					uploadFile(f)
				})
			}
		}

		fileInputRef.current = input
	}, [ props.id ])

	const onKeyEvent = (e: KeyboardEvent) => {
		if (e.code === 'Enter') {
			// Close the dialog only if all files were uploaded.
			// We can't use file state + progress because the state is unreliable! 
			if (currentUploads.current === 0) {
				props.setVisible(false)
			} else {
				// The user can wait up to 400 milliseconds. Otherwise something is wrong and the user should be notified about that
				setTimeout(() => {
					if (currentUploads.current === 0) {
						props.setVisible(false)
					} else {
						notify("Dateien wurden noch nicht vollständig hochgeladen", "warning", "light")
					}
				}, 400)
			}
		}
	}

	useEffect(() => {
		if(props.visible) window.addEventListener("keydown", onKeyEvent)
		else window.removeEventListener("keydown", onKeyEvent)

		return () => {
			window.removeEventListener("keydown", onKeyEvent)
		}
	}, [ props.visible ])

	return (
		<GenericModal
			visible={props.visible}
			setVisible={props.setVisible}
			title='Dateien hochladen'
		>
			<div id="drop-container" data-dragging={isDragging ? "true" : "false"}
				onDrop={(e) => {
					e.preventDefault()

					if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
						const droppedFiles = Array.from(e.dataTransfer.files).map(i => { return {progress: 0, name: i.name, extension: getFileExtension(i.name), size: i.size, file: i, id: (Math.random() + 1).toString(36).substring(2) }  })
						droppedFiles.forEach(f => {
							uploadFile(f)
						})
					}
				}}
				onDragEnter={() => setDragging(true)}
				onDragLeave={() => setDragging(false)}
				onDragOver={(e) => e.preventDefault()}
				onClick={() => fileInputRef.current?.click() }
			>

				{ files.length === 0 && 
					<div className="drop-info">
						<img 
							src={'/static/info-outlined.svg'} 
							style={{ width: "30px", alignSelf: "center", marginRight: "5px" }}
						/>
						<span>Ziehe Dateien in das Feld oder wähle welche von deinem Gerät aus</span>
					</div> 
				}

				{files.map(f => 
					<div className="file" data-progress={Math.floor( f.progress / 20)} key={f.id} /* style={{ background: "linear-gradient(to right, black " + f.progress + "%, rgb(165, 165, 165) " + (-2 + f.progress) +"%)" }} */ >
						<span className="file-prop" style={{ width: "40%" }}> { f.name } </span>
						<span className="file-prop" style={{ width: "15%" }}> { f.extension } </span>
						<span className="file-prop" style={{ width: "15%" }}> { getFileSizeInMb(f.size) } </span>
						<span className="file-prop" style={{ width: "15%" }}> { f.progress === 100 ? "Hochgeladen" : "..." } </span>
					</div>	
				)}

			</div>
				
		</GenericModal>
	)
}

function getFileExtension(input: string): string {
	return (/(?:\.([^.]+))?$/.exec(input) ?? ["-", "-"])[1] ?? "-"
}
function getFileSizeInMb(size: number): string {
	return (size / (1024 * 1024)).toFixed(2).toString() + "mb"
}

type CustomFile = {
	// Progress betweeen 0 and 100
	progress: number
	name: string
	extension: string
	size: number
	file: File

	id: string
}

export type UploadDialogProps = {
	setVisible: (visible: boolean) => void
	visible: boolean

	/** ID of the FileUploadRequest  */
	id: number
	/** File types to acceppt in the file picker dialog */
	accept: string
}