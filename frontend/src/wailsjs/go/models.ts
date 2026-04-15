export namespace config {
	
	export class ModelEntry {
	    model: string;
	    target: string;
	    alias?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.target = source["target"];
	        this.alias = source["alias"];
	    }
	}
	export class RouteRuleConfig {
	    name: string;
	    matchType: string;
	    pattern: string;
	    providerId: string;
	    targetModel: string;
	    priority: number;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RouteRuleConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.matchType = source["matchType"];
	        this.pattern = source["pattern"];
	        this.providerId = source["providerId"];
	        this.targetModel = source["targetModel"];
	        this.priority = source["priority"];
	        this.enabled = source["enabled"];
	    }
	}

}
