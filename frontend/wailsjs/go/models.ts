export namespace die {
	
	export class FavoriteEntry {
	    command: string;
	    label: string;
	    icon?: string;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	    }
	}
	export class HistoryEntry {
	    command: string;
	    timestamp: number;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.timestamp = source["timestamp"];
	        this.success = source["success"];
	    }
	}
	export class Intent {
	    action: string;
	    query?: string;
	    filters?: Record<string, string>;
	    target?: string;
	    params?: Record<string, string>;
	    raw_command: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new Intent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.query = source["query"];
	        this.filters = source["filters"];
	        this.target = source["target"];
	        this.params = source["params"];
	        this.raw_command = source["raw_command"];
	        this.confidence = source["confidence"];
	    }
	}
	export class Suggestion {
	    title: string;
	    description: string;
	    category: string;
	    icon?: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new Suggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.icon = source["icon"];
	        this.score = source["score"];
	    }
	}

}

export namespace disk {
	
	export class CHSGeometry {
	    cylinders: number;
	    heads: number;
	    sectorsPerTrack: number;
	
	    static createFrom(source: any = {}) {
	        return new CHSGeometry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cylinders = source["cylinders"];
	        this.heads = source["heads"];
	        this.sectorsPerTrack = source["sectorsPerTrack"];
	    }
	}
	export class ContainerInfo {
	    fileName: string;
	    filePath: string;
	    format: string;
	    diskType: string;
	    virtualSize: number;
	    physicalSize: number;
	    creatorApp: string;
	    creatorVersion: string;
	    creatorHostOS: string;
	    uniqueID: string;
	    checksumValid: boolean;
	    readonlyMode: boolean;
	    headerOffset?: number;
	    batOffset?: number;
	    batEntrySize?: number;
	    blockSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new ContainerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.filePath = source["filePath"];
	        this.format = source["format"];
	        this.diskType = source["diskType"];
	        this.virtualSize = source["virtualSize"];
	        this.physicalSize = source["physicalSize"];
	        this.creatorApp = source["creatorApp"];
	        this.creatorVersion = source["creatorVersion"];
	        this.creatorHostOS = source["creatorHostOS"];
	        this.uniqueID = source["uniqueID"];
	        this.checksumValid = source["checksumValid"];
	        this.readonlyMode = source["readonlyMode"];
	        this.headerOffset = source["headerOffset"];
	        this.batOffset = source["batOffset"];
	        this.batEntrySize = source["batEntrySize"];
	        this.blockSize = source["blockSize"];
	    }
	}
	export class DiskInfo {
	    filePath: string;
	    fileName: string;
	    fileSize: number;
	    virtualSize: number;
	    format: string;
	    diskType: string;
	    creatorApp: string;
	    creatorVersion: string;
	    creatorHostOS: string;
	    uniqueID: string;
	    checksumValid: boolean;
	    blockSize: number;
	    maxBATEntries: number;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.fileSize = source["fileSize"];
	        this.virtualSize = source["virtualSize"];
	        this.format = source["format"];
	        this.diskType = source["diskType"];
	        this.creatorApp = source["creatorApp"];
	        this.creatorVersion = source["creatorVersion"];
	        this.creatorHostOS = source["creatorHostOS"];
	        this.uniqueID = source["uniqueID"];
	        this.checksumValid = source["checksumValid"];
	        this.blockSize = source["blockSize"];
	        this.maxBATEntries = source["maxBATEntries"];
	        this.warnings = source["warnings"];
	    }
	}
	export class FSInfo {
	    filesystemType: string;
	    volumeUUID: string;
	    volumeLabel: string;
	    state: string;
	    mountCount: number;
	    maxMounts: number;
	    lastMountedPath: string;
	    lastWriteTime: string;
	    blockSize: number;
	    totalBlocks: number;
	    freeBlocks: number;
	    totalInodes: number;
	    freeInodes: number;
	    blockGroups: number;
	    featureFlags: string[];
	    superblockValid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FSInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filesystemType = source["filesystemType"];
	        this.volumeUUID = source["volumeUUID"];
	        this.volumeLabel = source["volumeLabel"];
	        this.state = source["state"];
	        this.mountCount = source["mountCount"];
	        this.maxMounts = source["maxMounts"];
	        this.lastMountedPath = source["lastMountedPath"];
	        this.lastWriteTime = source["lastWriteTime"];
	        this.blockSize = source["blockSize"];
	        this.totalBlocks = source["totalBlocks"];
	        this.freeBlocks = source["freeBlocks"];
	        this.totalInodes = source["totalInodes"];
	        this.freeInodes = source["freeInodes"];
	        this.blockGroups = source["blockGroups"];
	        this.featureFlags = source["featureFlags"];
	        this.superblockValid = source["superblockValid"];
	    }
	}
	export class PartitionInfo {
	    index: number;
	    label: string;
	    filesystem: string;
	    startLBA: number;
	    endLBA: number;
	    totalSectors: number;
	    sizeBytes: number;
	    bootable: boolean;
	    active: boolean;
	    isMounted: boolean;
	    isUnallocated: boolean;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new PartitionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.label = source["label"];
	        this.filesystem = source["filesystem"];
	        this.startLBA = source["startLBA"];
	        this.endLBA = source["endLBA"];
	        this.totalSectors = source["totalSectors"];
	        this.sizeBytes = source["sizeBytes"];
	        this.bootable = source["bootable"];
	        this.active = source["active"];
	        this.isMounted = source["isMounted"];
	        this.isUnallocated = source["isUnallocated"];
	        this.status = source["status"];
	    }
	}
	export class GeometryInfo {
	    totalSectors: number;
	    logicalSectorSize: number;
	    physicalSectorSize: number;
	    partitionScheme: string;
	    diskGUID?: string;
	    diskSignature?: string;
	    chs: CHSGeometry;
	
	    static createFrom(source: any = {}) {
	        return new GeometryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalSectors = source["totalSectors"];
	        this.logicalSectorSize = source["logicalSectorSize"];
	        this.physicalSectorSize = source["physicalSectorSize"];
	        this.partitionScheme = source["partitionScheme"];
	        this.diskGUID = source["diskGUID"];
	        this.diskSignature = source["diskSignature"];
	        this.chs = this.convertValues(source["chs"], CHSGeometry);
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
	export class DiskInfoResponse {
	    container: ContainerInfo;
	    geometry: GeometryInfo;
	    partitions: PartitionInfo[];
	    fsInfo?: FSInfo;
	
	    static createFrom(source: any = {}) {
	        return new DiskInfoResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.container = this.convertValues(source["container"], ContainerInfo);
	        this.geometry = this.convertValues(source["geometry"], GeometryInfo);
	        this.partitions = this.convertValues(source["partitions"], PartitionInfo);
	        this.fsInfo = this.convertValues(source["fsInfo"], FSInfo);
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
	
	
	export class Partition {
	    index: number;
	    start: number;
	    end: number;
	    size: number;
	    type: string;
	    filesystem: string;
	    bootable: boolean;
	    label: string;
	    active: boolean;
	    hasContent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Partition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.size = source["size"];
	        this.type = source["type"];
	        this.filesystem = source["filesystem"];
	        this.bootable = source["bootable"];
	        this.label = source["label"];
	        this.active = source["active"];
	        this.hasContent = source["hasContent"];
	    }
	}
	export class OpenResult {
	    info: DiskInfo;
	    partitions: Partition[];
	    activePartition?: Partition;
	    rootPath: string;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new OpenResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.info = this.convertValues(source["info"], DiskInfo);
	        this.partitions = this.convertValues(source["partitions"], Partition);
	        this.activePartition = this.convertValues(source["activePartition"], Partition);
	        this.rootPath = source["rootPath"];
	        this.warnings = source["warnings"];
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
	
	
	export class ValidationResult {
	    path: string;
	    filesize: number;
	    format: string;
	    valid: boolean;
	    warnings: string[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.filesize = source["filesize"];
	        this.format = source["format"];
	        this.valid = source["valid"];
	        this.warnings = source["warnings"];
	        this.errors = source["errors"];
	    }
	}

}

export namespace grammar {
	
	export class GrammarResponse {
	    grammar: string;
	    language: string;
	    extension: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new GrammarResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.grammar = source["grammar"];
	        this.language = source["language"];
	        this.extension = source["extension"];
	        this.source = source["source"];
	    }
	}

}

export namespace intelligence {
	
	export class Finding {
	    level: string;
	    category: string;
	    title: string;
	    description: string;
	    source: string;
	    path: string;
	    // Go type: time
	    timestamp: any;
	    formatted_ts: string;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.source = source["source"];
	        this.path = source["path"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.formatted_ts = source["formatted_ts"];
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

export namespace memory {
	
	export class Socket {
	    type: string;
	    local_addr: string;
	    remote_addr: string;
	    state: string;
	
	    static createFrom(source: any = {}) {
	        return new Socket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.local_addr = source["local_addr"];
	        this.remote_addr = source["remote_addr"];
	        this.state = source["state"];
	    }
	}
	export class ProcessInfo {
	    pid: number;
	    ppid: number;
	    name: string;
	    path: string;
	    command_line: string;
	    username: string;
	    create_time: string;
	    open_sockets: Socket[];
	    is_suspicious: boolean;
	    flag_reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.ppid = source["ppid"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.command_line = source["command_line"];
	        this.username = source["username"];
	        this.create_time = source["create_time"];
	        this.open_sockets = this.convertValues(source["open_sockets"], Socket);
	        this.is_suspicious = source["is_suspicious"];
	        this.flag_reason = source["flag_reason"];
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
	
	export class YaraMatch {
	    rule_name: string;
	    pid: number;
	    process_name: string;
	    matched_data: string;
	    severity: string;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new YaraMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rule_name = source["rule_name"];
	        this.pid = source["pid"];
	        this.process_name = source["process_name"];
	        this.matched_data = source["matched_data"];
	        this.severity = source["severity"];
	        this.tags = source["tags"];
	    }
	}

}

export namespace platform {
	
	export class Bookmark {
	    id: string;
	    target_id: string;
	    path: string;
	    label: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Bookmark(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.target_id = source["target_id"];
	        this.path = source["path"];
	        this.label = source["label"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class Gateway {
	
	
	    static createFrom(source: any = {}) {
	        return new Gateway(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class HistoryEntry {
	    target_id: string;
	    partition: string;
	    path: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_id = source["target_id"];
	        this.partition = source["partition"];
	        this.path = source["path"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	export class WorkspaceTarget {
	    id: string;
	    image_path: string;
	    format: string;
	    partitions: string[];
	    is_encrypted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.image_path = source["image_path"];
	        this.format = source["format"];
	        this.partitions = source["partitions"];
	        this.is_encrypted = source["is_encrypted"];
	    }
	}
	export class WorkspaceState {
	    id: string;
	    name: string;
	    targets: WorkspaceTarget[];
	    active_target_id: string;
	    active_partition: string;
	    bookmarks: Bookmark[];
	    opened_tabs: string[];
	    active_tab_index: number;
	    history: HistoryEntry[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.targets = this.convertValues(source["targets"], WorkspaceTarget);
	        this.active_target_id = source["active_target_id"];
	        this.active_partition = source["active_partition"];
	        this.bookmarks = this.convertValues(source["bookmarks"], Bookmark);
	        this.opened_tabs = source["opened_tabs"];
	        this.active_tab_index = source["active_tab_index"];
	        this.history = this.convertValues(source["history"], HistoryEntry);
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

export namespace sigma {
	
	export class Alert {
	    rule_title: string;
	    level: string;
	    description: string;
	    tags: string[];
	    log_source: string;
	    path: string;
	    timestamp: string;
	    matched_log: string;
	
	    static createFrom(source: any = {}) {
	        return new Alert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rule_title = source["rule_title"];
	        this.level = source["level"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.log_source = source["log_source"];
	        this.path = source["path"];
	        this.timestamp = source["timestamp"];
	        this.matched_log = source["matched_log"];
	    }
	}

}

export namespace storage {
	
	export class ProvenanceEntry {
	    // Go type: time
	    timestamp: any;
	    actor: string;
	    action: string;
	    target: string;
	    details: string;
	    session_id: string;
	
	    static createFrom(source: any = {}) {
	        return new ProvenanceEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.actor = source["actor"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.details = source["details"];
	        this.session_id = source["session_id"];
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
	export class Timestamps {
	    // Go type: time
	    created: any;
	    // Go type: time
	    modified: any;
	    // Go type: time
	    accessed: any;
	    // Go type: time
	    mft_modified: any;
	
	    static createFrom(source: any = {}) {
	        return new Timestamps(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = this.convertValues(source["created"], null);
	        this.modified = this.convertValues(source["modified"], null);
	        this.accessed = this.convertValues(source["accessed"], null);
	        this.mft_modified = this.convertValues(source["mft_modified"], null);
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
	export class UniversalFileNode {
	    name: string;
	    path: string;
	    is_dir: boolean;
	    size: number;
	    timestamps: Timestamps;
	    permissions: string;
	    owner_id: number;
	    group_id: number;
	    file_id: number;
	    attributes: string[];
	    is_deleted: boolean;
	    stream_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new UniversalFileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.is_dir = source["is_dir"];
	        this.size = source["size"];
	        this.timestamps = this.convertValues(source["timestamps"], Timestamps);
	        this.permissions = source["permissions"];
	        this.owner_id = source["owner_id"];
	        this.group_id = source["group_id"];
	        this.file_id = source["file_id"];
	        this.attributes = source["attributes"];
	        this.is_deleted = source["is_deleted"];
	        this.stream_name = source["stream_name"];
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

export namespace timeline {
	
	export class TimelineEntry {
	    id: string;
	    // Go type: time
	    timestamp: any;
	    source: string;
	    event_type: string;
	    title: string;
	    description: string;
	    path?: string;
	    data?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new TimelineEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.source = source["source"];
	        this.event_type = source["event_type"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.path = source["path"];
	        this.data = source["data"];
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

export namespace ui {
	
	export class ExportReportRequest {
	    caseDir: string;
	    examinerName: string;
	    outputPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportReportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.caseDir = source["caseDir"];
	        this.examinerName = source["examinerName"];
	        this.outputPath = source["outputPath"];
	    }
	}
	export class ExportReportResponse {
	    reportPath: string;
	    success: boolean;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportReportResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reportPath = source["reportPath"];
	        this.success = source["success"];
	        this.errorMessage = source["errorMessage"];
	    }
	}
	export class IngestRequest {
	    source_path: string;
	    case_dir: string;
	    passphrase: string;
	    case_id: string;
	    actor: string;
	
	    static createFrom(source: any = {}) {
	        return new IngestRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_path = source["source_path"];
	        this.case_dir = source["case_dir"];
	        this.passphrase = source["passphrase"];
	        this.case_id = source["case_id"];
	        this.actor = source["actor"];
	    }
	}
	export class NodeDTO {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new NodeDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	    }
	}
	export class PartitionDTO {
	    index: number;
	    type: string;
	    start: number;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new PartitionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.type = source["type"];
	        this.start = source["start"];
	        this.size = source["size"];
	    }
	}
	export class PreviewResponse {
	    content: string;
	    isBinary: boolean;
	    isTruncated: boolean;
	    size: number;
	    fileName: string;
	    extension: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.isBinary = source["isBinary"];
	        this.isTruncated = source["isTruncated"];
	        this.size = source["size"];
	        this.fileName = source["fileName"];
	        this.extension = source["extension"];
	    }
	}
	export class RecentFile {
	    path: string;
	    name: string;
	    size: number;
	    openedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new RecentFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.openedAt = source["openedAt"];
	    }
	}
	export class SQLQueryResult {
	    columns: string[];
	    rows: any[];
	    count: number;
	    time_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new SQLQueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.count = source["count"];
	        this.time_ms = source["time_ms"];
	    }
	}
	export class SessionInfo {
	    id: string;
	    state: string;
	    image_path: string;
	    file_name: string;
	    format: string;
	    total_size: number;
	    is_read_only: boolean;
	    partition_count: number;
	    filesystem: string;
	    provenance: storage.ProvenanceEntry[];
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.state = source["state"];
	        this.image_path = source["image_path"];
	        this.file_name = source["file_name"];
	        this.format = source["format"];
	        this.total_size = source["total_size"];
	        this.is_read_only = source["is_read_only"];
	        this.partition_count = source["partition_count"];
	        this.filesystem = source["filesystem"];
	        this.provenance = this.convertValues(source["provenance"], storage.ProvenanceEntry);
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
	export class VaultResult {
	    manifest?: vault.Manifest;
	    audit_valid: boolean;
	    audit_count: number;
	    chunks_ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VaultResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifest = this.convertValues(source["manifest"], vault.Manifest);
	        this.audit_valid = source["audit_valid"];
	        this.audit_count = source["audit_count"];
	        this.chunks_ok = source["chunks_ok"];
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
	export class VerificationResult {
	    valid: boolean;
	    audit_count: number;
	    manifest_present: boolean;
	    source_hash: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new VerificationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.audit_count = source["audit_count"];
	        this.manifest_present = source["manifest_present"];
	        this.source_hash = source["source_hash"];
	        this.message = source["message"];
	    }
	}

}

export namespace vault {
	
	export class Argon2Params {
	    memory_kb: number;
	    iterations: number;
	    parallelism: number;
	    salt_len: number;
	    key_len: number;
	
	    static createFrom(source: any = {}) {
	        return new Argon2Params(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memory_kb = source["memory_kb"];
	        this.iterations = source["iterations"];
	        this.parallelism = source["parallelism"];
	        this.salt_len = source["salt_len"];
	        this.key_len = source["key_len"];
	    }
	}
	export class Manifest {
	    magic: string;
	    format_version: number;
	    vault_id: string;
	    // Go type: time
	    created_at: any;
	    kdf_params: Argon2Params;
	    salt: number[];
	    cipher: string;
	    chunk_size: number;
	    wrapped_dek: number[];
	    wrapped_dek_nonce: number[];
	    total_size_bytes: number;
	    total_chunks: number;
	    source_sha256: string;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.magic = source["magic"];
	        this.format_version = source["format_version"];
	        this.vault_id = source["vault_id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.kdf_params = this.convertValues(source["kdf_params"], Argon2Params);
	        this.salt = source["salt"];
	        this.cipher = source["cipher"];
	        this.chunk_size = source["chunk_size"];
	        this.wrapped_dek = source["wrapped_dek"];
	        this.wrapped_dek_nonce = source["wrapped_dek_nonce"];
	        this.total_size_bytes = source["total_size_bytes"];
	        this.total_chunks = source["total_chunks"];
	        this.source_sha256 = source["source_sha256"];
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

export namespace vfs {
	
	export class EntryMeta {
	    owner?: string;
	    group?: string;
	    permissions?: string;
	    mimeType?: string;
	
	    static createFrom(source: any = {}) {
	        return new EntryMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owner = source["owner"];
	        this.group = source["group"];
	        this.permissions = source["permissions"];
	        this.mimeType = source["mimeType"];
	    }
	}
	export class Entry {
	    name: string;
	    path: string;
	    size: number;
	    isDir: boolean;
	    modTime: number;
	    extension: string;
	    type: string;
	    targetPath?: string;
	    metadata?: EntryMeta;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.isDir = source["isDir"];
	        this.modTime = source["modTime"];
	        this.extension = source["extension"];
	        this.type = source["type"];
	        this.targetPath = source["targetPath"];
	        this.metadata = this.convertValues(source["metadata"], EntryMeta);
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

