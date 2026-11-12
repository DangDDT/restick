async function referencePng(assetId: string): Promise<Blob> {
  const response = await fetch(`/api/v1/admin/reference-assets/${encodeURIComponent(assetId)}`, {
    credentials: "same-origin",
    cache: "no-store",
    redirect: "error",
  });
  if (!response.ok) throw new Error(`Reference image request failed (${response.status}).`);
  const blob = await response.blob();
  if (!blob.size) throw new Error("Reference image is empty.");
  if (blob.type === "image/png") return blob;
  if (blob.type !== "image/jpeg" && blob.type !== "image/webp") {
    throw new Error("Reference image must be PNG, JPEG or WebP.");
  }

  const url = URL.createObjectURL(blob);
  const image = new Image();
  const canvas = document.createElement("canvas");
  try {
    image.src = url;
    await image.decode();
    if (!image.naturalWidth || !image.naturalHeight) throw new Error("Reference image could not be decoded.");
    canvas.width = image.naturalWidth;
    canvas.height = image.naturalHeight;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("Image conversion is unavailable.");
    context.drawImage(image, 0, 0);
    const { promise, resolve, reject } = Promise.withResolvers<Blob>();
    canvas.toBlob((png) => {
      if (png?.size && png.type === "image/png") resolve(png);
      else reject(new Error("PNG conversion failed."));
    }, "image/png");
    return await promise;
  } finally {
    image.removeAttribute("src");
    URL.revokeObjectURL(url);
    canvas.width = 0;
    canvas.height = 0;
  }
}

/** Call directly from a click: write must start before fetch/decode yields. */
export async function copyReferenceImage(assetId: string): Promise<void> {
  if (!window.isSecureContext || !navigator.clipboard?.write || typeof ClipboardItem === "undefined") {
    throw new Error("This browser cannot copy images. Open the private image to copy it manually.");
  }
  if (typeof ClipboardItem.supports === "function" && !ClipboardItem.supports("image/png")) {
    throw new Error("This browser cannot write PNG images to the clipboard.");
  }
  const png = referencePng(assetId);
  // A denied write can reject before the fetch; still observe that promise's rejection.
  void png.catch(() => {});
  await navigator.clipboard.write([new ClipboardItem({ "image/png": png })]);
}
