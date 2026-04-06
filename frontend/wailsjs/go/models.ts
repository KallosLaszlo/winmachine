export namespace backup {
	
	export class BackupStatus {
	    running: boolean;
	    lastSnapshot: string;
	    // Go type: time
	    lastTime: any;
	    progress: number;
	    currentFile: string;
	    filesTotal: number;
	    filesDone: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.lastSnapshot = source["lastSnapshot"];
	        this.lastTime = this.convertValues(source["lastTime"], null);
	        this.progress = source["progress"];
	        this.currentFile = source["currentFile"];
	        this.filesTotal = source["filesTotal"];
	        this.filesDone = source["filesDone"];
	        this.error = source["error"];
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
	export class FileInfo {
	    name: string;
	    relPath: string;
	    size: number;
	    modTime: number;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relPath = source["relPath"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.isDir = source["isDir"];
	    }
	}
	export class SnapshotMeta {
	    id: string;
	    status: string;
	    // Go type: time
	    timestamp: any;
	    sourceDirs: string[];
	    machineId: string;
	    fileCount: number;
	    totalSize: number;
	    linkedSize: number;
	    copiedSize: number;
	    duration: string;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.sourceDirs = source["sourceDirs"];
	        this.machineId = source["machineId"];
	        this.fileCount = source["fileCount"];
	        this.totalSize = source["totalSize"];
	        this.linkedSize = source["linkedSize"];
	        this.copiedSize = source["copiedSize"];
	        this.duration = source["duration"];
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

export namespace config {
	
	export class RetentionPolicy {
	    hourlyForHours: number;
	    dailyForDays: number;
	    weeklyForWeeks: number;
	    monthlyForMonths: number;
	
	    static createFrom(source: any = {}) {
	        return new RetentionPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hourlyForHours = source["hourlyForHours"];
	        this.dailyForDays = source["dailyForDays"];
	        this.weeklyForWeeks = source["weeklyForWeeks"];
	        this.monthlyForMonths = source["monthlyForMonths"];
	    }
	}
	export class SMBShareConfig {
	    server: string;
	    share: string;
	    username: string;
	    password: string;
	    domain: string;
	    drive: string;
	
	    static createFrom(source: any = {}) {
	        return new SMBShareConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.share = source["share"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.domain = source["domain"];
	        this.drive = source["drive"];
	    }
	}
	export class Config {
	    sourceDirs: string[];
	    targetDir: string;
	    targetType: string;
	    smbTarget: SMBShareConfig;
	    scheduleInterval: string;
	    retention: RetentionPolicy;
	    autoStart: boolean;
	    excludePatterns: string[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceDirs = source["sourceDirs"];
	        this.targetDir = source["targetDir"];
	        this.targetType = source["targetType"];
	        this.smbTarget = this.convertValues(source["smbTarget"], SMBShareConfig);
	        this.scheduleInterval = source["scheduleInterval"];
	        this.retention = this.convertValues(source["retention"], RetentionPolicy);
	        this.autoStart = source["autoStart"];
	        this.excludePatterns = source["excludePatterns"];
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

