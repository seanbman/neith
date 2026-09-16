import { addEventListeners, parseEventListeners } from "./events";
import type { Dispatch, DispatchFunctions } from "./neith_types";
import { Fun, PROTOCOL_VERSION } from "./neith_types";
import { encodeMessage } from "./protocol";
import { emitHook } from "./hooks";
import { applyClass, applyCustom, applyDOM, applyMutate, applyRender } from "./render";

/** API executes the finite set of browser operations represented by protocol v1. */
export class API {
    private ws: WebSocket | null = null;
    constructor(ws: WebSocket) { this.ws = ws; }
    public Process(d: Dispatch) {
        if (!d || d.v !== PROTOCOL_VERSION) { const version=d&&typeof d.v==="number"?d.v:"missing"; emitHook("error",{dispatch:d,error:`unsupported protocol version: ${version}`}); return; }
        switch(d.function){
            case Fun.REDIRECT: window.location.href=d.redirect.url; break;
            default: if(!this.funs[d.function]){this.Error(d,"function not found: "+d.function);break} const result=this.funs[d.function](d);if(result)this.Dispatch(result);break;
        }
    }
    private Dispatch=(data:Dispatch|void)=>{if(!data)return;if(!this.ws)throw new Error("ws: not connected to server...");this.ws.send(JSON.stringify(encodeMessage(data)))};
    private Error=(d:Dispatch,message:string)=>{d.function=Fun.ERROR;d.error={message};emitHook("error",{dispatch:d,error:message});this.Dispatch(d)};
    private funs:DispatchFunctions={
        ping:(d)=>{d.ping.client=true;return d},
        render:(d)=>{emitHook("beforeRender",{dispatch:d});const elem=applyRender(d,this.Error);if(!elem)return;const dispatch=parseEventListeners(elem,d);addEventListeners(dispatch,this.Dispatch,this.Error);emitHook("afterRender",{dispatch,element:elem})},
        mutate:(d)=>applyMutate(d,this.Error),
        // Transitional v1 compatibility. New server helpers emit mutate.
        class:(d)=>applyClass(d,this.Error), dom:(d)=>applyDOM(d,this.Error), custom:(d)=>applyCustom(d,this.Error),
    };
}
