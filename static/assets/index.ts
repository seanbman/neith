import { Socket } from "./socket";

/**
 * Browser bundle entrypoint.
 *
 * Importing this file creates the websocket connection immediately. esbuild
 * starts from this module, follows its imports, and produces the single
 * minified browser file served by the examples and README setup.
 */
new Socket();
