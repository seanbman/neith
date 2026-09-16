import { Dispatch, EventTargetData, FnClass, FnCustom, FnDOM, FnError, FnEventListener, FnMutate, FnPing, FnRedirect, FnRender, Fun, PROTOCOL_VERSION, ProtocolMessage } from "./neith_types";

const envelopeKeys = new Set(["v","type","id","key","conn_id","handler_id","action","label","payload"]);
const mutationKeys = new Set(["operation","names","name","value"]);
const mutationOperations = new Set(["addClass","removeClass","setAttribute","removeAttribute","setStyle","removeStyle","setText","setValue","focus","blur","scrollIntoView","disable","enable"]);
const payloadKeys: Record<Fun, Set<string>> = {
    [Fun.PING]:new Set(["server","client"]), [Fun.RENDER]:new Set(["target_id","tag","inner","outer","append","prepend","remove","html","event_listeners"]),
    [Fun.MUTATE]:new Set(["target_id","mutations"]), [Fun.CLASS]:new Set(["target_id","remove","names"]), [Fun.DOM]:new Set(["target_id","operation","name","value"]),
    [Fun.CUSTOM]:new Set(["function","data","result"]), [Fun.REDIRECT]:new Set(["url"]), [Fun.EVENT]:new Set(["id","target_id","on","action","method","form_data","data","uploads","submitter"]), [Fun.ERROR]:new Set(["message"]),
};
const knownTypes = new Set<string>(Object.values(Fun));

export function decodeMessage(input:unknown):Dispatch {
    const message=object(input,"protocol envelope"); rejectUnknown(message,envelopeKeys,"protocol envelope");
    if(message.v!==PROTOCOL_VERSION) throw new Error(`unsupported protocol version: ${typeof message.v==="number"?message.v:"missing"}`);
    if(typeof message.type!=="string"||!knownTypes.has(message.type)) throw new Error(`unknown protocol type: ${String(message.type)}`);
    if(!("payload" in message)||message.payload===null) throw new Error("protocol payload is required");
    const type=message.type as Fun, payload=object(message.payload,`${type} payload`); rejectUnknown(payload,payloadKeys[type],`${type} payload`); validatePayload(type,payload);
    const dispatch=emptyDispatch(type); dispatch.id=optionalString(message.id); dispatch.key=optionalString(message.key); dispatch.conn_id=optionalString(message.conn_id); dispatch.handler_id=optionalString(message.handler_id); dispatch.action=optionalString(message.action); dispatch.label=optionalString(message.label);
    switch(type){case Fun.PING:dispatch.ping=payload as FnPing;break;case Fun.RENDER:dispatch.render=payload as FnRender;break;case Fun.MUTATE:dispatch.mutate=payload as FnMutate;break;case Fun.CLASS:dispatch.class=payload as FnClass;break;case Fun.DOM:dispatch.dom=payload as FnDOM;break;case Fun.CUSTOM:dispatch.custom=payload as FnCustom;break;case Fun.REDIRECT:dispatch.redirect=payload as FnRedirect;break;case Fun.EVENT:dispatch.event=payload as FnEventListener;break;case Fun.ERROR:dispatch.error=payload as FnError;break;}
    return dispatch;
}

export function encodeMessage(dispatch:Dispatch):ProtocolMessage {
    if(dispatch.v!==PROTOCOL_VERSION) throw new Error(`unsupported protocol version: ${dispatch.v}`); if(!knownTypes.has(dispatch.function)) throw new Error(`unknown protocol type: ${String(dispatch.function)}`);
    let payload:unknown; switch(dispatch.function){case Fun.PING:payload=dispatch.ping;break;case Fun.RENDER:payload=dispatch.render;break;case Fun.MUTATE:payload=dispatch.mutate;break;case Fun.CLASS:payload=dispatch.class;break;case Fun.DOM:payload=dispatch.dom;break;case Fun.CUSTOM:payload=dispatch.custom;break;case Fun.REDIRECT:payload=dispatch.redirect;break;case Fun.EVENT:payload=dispatch.event;break;case Fun.ERROR:payload=dispatch.error;break;}
    validatePayload(dispatch.function,object(payload,`${dispatch.function} payload`));
    return {v:PROTOCOL_VERSION,type:dispatch.function,...(dispatch.id?{id:dispatch.id}:{}),...(dispatch.key?{key:dispatch.key}:{}),...(dispatch.conn_id?{conn_id:dispatch.conn_id}:{}),...(dispatch.handler_id?{handler_id:dispatch.handler_id}:{}),...(dispatch.action?{action:dispatch.action}:{}),...(dispatch.label?{label:dispatch.label}:{}),payload};
}

function validatePayload(type:Fun,value:Record<string,unknown>){
    switch(type){
        case Fun.PING:boolean(value.server,"ping.server");boolean(value.client,"ping.client");break;
        case Fun.RENDER:string(value.html,"render.html");if(value.target_id!==undefined)string(value.target_id,"render.target_id");if(value.tag!==undefined)string(value.tag,"render.tag");for(const key of ["inner","outer","append","prepend","remove"]){if(value[key]!==undefined)boolean(value[key],`render.${key}`)}if(value.event_listeners!==undefined&&!Array.isArray(value.event_listeners))throw new Error("render.event_listeners must be an array");break;
        case Fun.MUTATE:string(value.target_id,"mutate.target_id");if(!Array.isArray(value.mutations)||value.mutations.length===0)throw new Error("mutate.mutations must be a non-empty array");for(const raw of value.mutations){const mutation=object(raw,"mutation");rejectUnknown(mutation,mutationKeys,"mutation");string(mutation.operation,"mutation.operation");if(!mutationOperations.has(mutation.operation))throw new Error(`mutation operation not allowed: ${mutation.operation}`);if(mutation.names!==undefined&&(!Array.isArray(mutation.names)||mutation.names.some((n)=>typeof n!=="string")))throw new Error("mutation.names must be a string array");if(mutation.name!==undefined)string(mutation.name,"mutation.name");if(mutation.value!==undefined)string(mutation.value,"mutation.value");}break;
        case Fun.CLASS:string(value.target_id,"class.target_id");boolean(value.remove,"class.remove");if(!Array.isArray(value.names)||value.names.some((n)=>typeof n!=="string"))throw new Error("class.names must be a string array");break;
        case Fun.DOM:string(value.target_id,"dom.target_id");string(value.operation,"dom.operation");if(value.name!==undefined)string(value.name,"dom.name");if(value.value!==undefined)string(value.value,"dom.value");break;
        case Fun.CUSTOM:string(value.function,"custom.function");break;case Fun.REDIRECT:string(value.url,"redirect.url");break;case Fun.EVENT:string(value.id,"event.id");string(value.on,"event.on");if(value.target_id!==undefined)string(value.target_id,"event.target_id");break;case Fun.ERROR:string(value.message,"error.message");break;
    }
}
function emptyDispatch(type:Fun):Dispatch{return{v:PROTOCOL_VERSION,function:type,id:"",key:"",conn_id:"",handler_id:"",action:"",label:"",event:{} as FnEventListener,ping:{server:false,client:false},render:{target_id:"",tag:"",inner:false,outer:false,append:false,prepend:false,remove:false,html:"",event_listeners:[]},mutate:{target_id:"",mutations:[]},class:{target_id:"",remove:false,names:[]},dom:{target_id:"",operation:"",name:"",value:""},redirect:{url:""},custom:{function:"",data:{},result:{}},error:{message:""}}}
function object(value:unknown,name:string):Record<string,any>{if(value===null||typeof value!=="object"||Array.isArray(value))throw new Error(`${name} must be an object`);return value as Record<string,any>}
function rejectUnknown(value:Record<string,unknown>,allowed:Set<string>,name:string){for(const key of Object.keys(value)){if(!allowed.has(key))throw new Error(`${name} contains unknown field: ${key}`)}}
function string(value:unknown,name:string):asserts value is string{if(typeof value!=="string")throw new Error(`${name} must be a string`)}
function boolean(value:unknown,name:string):asserts value is boolean{if(typeof value!=="boolean")throw new Error(`${name} must be a boolean`)}
function optionalString(value:unknown):string{if(value===undefined)return"";if(typeof value!=="string")throw new Error("protocol metadata must be strings");return value}
