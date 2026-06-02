/** Portrait phones and landscape phones (width alone misses iPhone landscape at 844px). */
const MOBILE_MEDIA = '(max-width: 767px), (max-width: 932px) and (max-height: 500px)';

class ViewportStore {
	isMobile = $state(
		typeof window !== 'undefined' ? window.matchMedia(MOBILE_MEDIA).matches : false
	);
	#bound = false;

	bind() {
		if (this.#bound || typeof window === 'undefined') return () => {};
		this.#bound = true;
		const mql = window.matchMedia(MOBILE_MEDIA);
		const update = () => {
			this.isMobile = mql.matches;
		};
		update();
		mql.addEventListener('change', update);
		return () => {
			mql.removeEventListener('change', update);
			this.#bound = false;
		};
	}
}

export const viewport = new ViewportStore();
