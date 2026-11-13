async function imagePng(url: string, label: string): Promise<Blob> {
  const response = await fetch(url, {
    credentials: "same-origin",
    cache: "no-store",
    redirect: "error",
  });
  if (!response.ok) throw new Error(`${label} request failed (${response.status}).`);
  const blob = await response.blob();
  if (!blob.size) throw new Error(`${label} is empty.`);
  if (blob.type === "image/png") return blob;
  if (blob.type !== "image/jpeg" && blob.type !== "image/webp") {
    throw new Error(`${label} must be PNG, JPEG or WebP.`);
  }

  const blobUrl = URL.createObjectURL(blob);
  const image = new Image();
  const canvas = document.createElement("canvas");
  try {
    image.src = blobUrl;
    await image.decode();
    if (!image.naturalWidth || !image.naturalHeight) throw new Error(`${label} could not be decoded.`);
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
    URL.revokeObjectURL(blobUrl);
    canvas.width = 0;
    canvas.height = 0;
  }
}

/**
 * Copy a same-origin image as PNG. Call directly from a click: the clipboard
 * write must start before fetch/decode yields.
 */
export async function copyImageURL(url: string, label = "Reference image"): Promise<void> {
  if (!window.isSecureContext || !navigator.clipboard?.write || typeof ClipboardItem === "undefined") {
    throw new Error("This browser cannot copy images. Open the private image to copy it manually.");
  }
  if (typeof ClipboardItem.supports === "function" && !ClipboardItem.supports("image/png")) {
    throw new Error("This browser cannot write PNG images to the clipboard.");
  }
  const png = imagePng(url, label);
  // A denied write can reject before the fetch; still observe that promise's rejection.
  void png.catch(() => {});
  await navigator.clipboard.write([new ClipboardItem({ "image/png": png })]);
}

export function copyReferenceImage(assetId: string): Promise<void> {
  return copyImageURL(`/api/v1/admin/reference-assets/${encodeURIComponent(assetId)}`);
}
