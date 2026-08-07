import { useEvidenceStore } from '../store/evidenceStore';

export async function openDisk(): Promise<void> {
  const { Main } = await import('./wails');

  // Step 1: Show native file dialog
  let filePath: string;
  try {
    filePath = await Main.OpenFileDialog();
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    window.alert('Failed to open file dialog:\n' + msg);
    return;
  }

  if (!filePath) return;

  // Step 2: Activate evidence session (calls StorageHandler.MountDisk — fast single open)
  await useEvidenceStore.getState().openEvidence(filePath);
}
