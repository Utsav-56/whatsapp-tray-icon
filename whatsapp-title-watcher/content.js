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

	const callback = async () => {
		const title = document.title;
		const match = title.match(/\((\d+)\)/);
		const count = match ? match[1] : "0";

		if (count !== lastCount) {
			lastCount = count;

			await trySendCount(count);
		}
	};

	const observer = new MutationObserver(callback);
	observer.observe(targetNode, { childList: true });

	// Run once initially
	callback();
}
