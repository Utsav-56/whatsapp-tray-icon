// Wait for the title element to be available in the DOM
const targetNode = document.querySelector("title");

console.log("WhatsApp Title Watcher loaded");

async function trySendCount(count) {
	try {
		return await chrome.runtime.sendMessage({ type: "set_count", count });
	} catch (error) {
		console.error("Failed to reach tray daemon:", error.message);
		return false;
	}
}

if (targetNode) {
	let lastCount = "";

	// Made the callback async to handle sequential async operations
	const callback = async () => {
		const title = document.title;
		const match = title.match(/\((\d+)\)/);
		const count = match ? match[1] : "0";

		// Only send an HTTP request if the count actually changed
		if (count !== lastCount) {
			lastCount = count;

			await trySendCount(count);
		}
	};

	// MutationObserver reacts to DOM modifications natively without loops
	const observer = new MutationObserver(callback);
	observer.observe(targetNode, { childList: true });

	// Run once initially
	callback();
}
