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

}

