import { DependencyList, EffectCallback, useEffect, useRef, useState } from "react";

/**
 * Custom hook that is run when a value of the dependencies changed. In contrast to a "normal"
 * useEffect this is not run after mounting and also not twice in strict mode.
 */
export const useEffectAfterMount = (cb: EffectCallback, dependencies: DependencyList | undefined) => {
	const initialMount = useRef(true)
  
	// We need some boilerpat code around this.
	// In strict mode react calls the useEffect twice!
	// This shouldn't be a probleme because by design we should "undo"
	// the operation in the cleanup function.
	// This is not possible here, because a setSteate cannot
	// be undone....
	// So we use a timeout of 50 seconds which should be enough
	let timeout = 0
	useEffect(() => {
		timeout = setTimeout(() => {
			if (!initialMount.current) {
				cb()
			} else {
				initialMount.current = false
			}
		}, 50)

		return () => {
			clearTimeout(timeout)
		};
	}, dependencies);
};