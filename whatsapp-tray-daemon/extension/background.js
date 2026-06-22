async function setCount(count) {
	const hosts = ["localhost", "127.0.0.1"];

	for (const host of hosts) {
		const url = new URL("set_count", `http://${host}:63845`);
		url.searchParams.set("count", String(count));

		try {
			const response = await fetch(url, { cache: "no-store" });
			if (response.ok) {
				return true;
			}
		} catch (error) {
			console.warn(`Failed to reach tray daemon at ${url}:`, error.message);
		}
	}

	return false;
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
	if (message?.type !== "set_count") {
		return false;
	}

	setCount(message.count)
		.then((ok) => sendResponse(ok))
		.catch((error) => {
			console.error("Failed to update tray daemon:", error);
			sendResponse(false);
		});

	return true;
});
