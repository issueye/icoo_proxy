export namespace main {
	
	export class ServerConfig {
	    Host: string;
	    Port: number;
	    ReadTimeoutSeconds: number;
	    WriteTimeoutSeconds: number;
	    ShutdownTimeoutSeconds: number;
	    APIKeys: string[];
	    AllowUnauthenticatedLocal: boolean;
	    ChainLogPath: string;
	    ChainLogBodies: boolean;
	    ChainLogMaxBodyBytes: number;
	    DefaultMaxTokens: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Host = source["Host"];
	        this.Port = source["Port"];
	        this.ReadTimeoutSeconds = source["ReadTimeoutSeconds"];
	        this.WriteTimeoutSeconds = source["WriteTimeoutSeconds"];
	        this.ShutdownTimeoutSeconds = source["ShutdownTimeoutSeconds"];
	        this.APIKeys = source["APIKeys"];
	        this.AllowUnauthenticatedLocal = source["AllowUnauthenticatedLocal"];
	        this.ChainLogPath = source["ChainLogPath"];
	        this.ChainLogBodies = source["ChainLogBodies"];
	        this.ChainLogMaxBodyBytes = source["ChainLogMaxBodyBytes"];
	        this.DefaultMaxTokens = source["DefaultMaxTokens"];
	    }
	}
	export class ServerProcessInfo {
	    running: boolean;
	    status: string;
	    pid: number;
	    executable: string;
	    working_directory: string;
	    data_dir: string;
	    listen_addr: string;
	    started_at: string;
	    args: string[];
	    log_path: string;
	    last_error: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.status = source["status"];
	        this.pid = source["pid"];
	        this.executable = source["executable"];
	        this.working_directory = source["working_directory"];
	        this.data_dir = source["data_dir"];
	        this.listen_addr = source["listen_addr"];
	        this.started_at = source["started_at"];
	        this.args = source["args"];
	        this.log_path = source["log_path"];
	        this.last_error = source["last_error"];
	    }
	}

}

