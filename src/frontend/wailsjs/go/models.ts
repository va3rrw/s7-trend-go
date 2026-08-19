export namespace backend {
	
	export class YAxisSettings {
	    name: string;
	    minimum: number;
	    maximum: number;
	    autoScale: boolean;
	
	    static createFrom(source: any = {}) {
	        return new YAxisSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.minimum = source["minimum"];
	        this.maximum = source["maximum"];
	        this.autoScale = source["autoScale"];
	    }
	}
	export class TagSettings {
	    id: number[];
	    name: string;
	    plcLink: string;
	    address: string;
	    dataType: string;
	    yAxis: string;
	    color: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TagSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.plcLink = source["plcLink"];
	        this.address = source["address"];
	        this.dataType = source["dataType"];
	        this.yAxis = source["yAxis"];
	        this.color = source["color"];
	        this.enabled = source["enabled"];
	    }
	}
	export class PlcLinkSettings {
	    name: string;
	    ipAddress: string;
	    rack: number;
	    slot: number;
	    isConnected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlcLinkSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ipAddress = source["ipAddress"];
	        this.rack = source["rack"];
	        this.slot = source["slot"];
	        this.isConnected = source["isConnected"];
	    }
	}
	export class AppSettings {
	    pollIntervalMs: number;
	    timeWindowSeconds: number;
	    interpolation: string;
	    plcLinks: PlcLinkSettings[];
	    tags: TagSettings[];
	    yAxes: YAxisSettings[];
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pollIntervalMs = source["pollIntervalMs"];
	        this.timeWindowSeconds = source["timeWindowSeconds"];
	        this.interpolation = source["interpolation"];
	        this.plcLinks = this.convertValues(source["plcLinks"], PlcLinkSettings);
	        this.tags = this.convertValues(source["tags"], TagSettings);
	        this.yAxes = this.convertValues(source["yAxes"], YAxisSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

