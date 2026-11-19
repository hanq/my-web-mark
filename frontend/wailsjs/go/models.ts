export namespace app {
	
	export class AccountInput {
	    themeId: number;
	    kind: string;
	    bookmarkId: number;
	    site: string;
	    username: string;
	    password: string;
	    url: string;
	    note: string;
	    tagIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new AccountInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeId = source["themeId"];
	        this.kind = source["kind"];
	        this.bookmarkId = source["bookmarkId"];
	        this.site = source["site"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.url = source["url"];
	        this.note = source["note"];
	        this.tagIds = source["tagIds"];
	    }
	}
	export class AccountSummary {
	    id: number;
	    themeId: number;
	    kind: string;
	    bookmarkId?: number;
	    site: string;
	    username: string;
	    hasPwd: boolean;
	    url: string;
	    note: string;
	    tags: store.Tag[];
	
	    static createFrom(source: any = {}) {
	        return new AccountSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.themeId = source["themeId"];
	        this.kind = source["kind"];
	        this.bookmarkId = source["bookmarkId"];
	        this.site = source["site"];
	        this.username = source["username"];
	        this.hasPwd = source["hasPwd"];
	        this.url = source["url"];
	        this.note = source["note"];
	        this.tags = this.convertValues(source["tags"], store.Tag);
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
	export class BookmarkInput {
	    themeId: number;
	    title: string;
	    url: string;
	    favicon: string;
	    note: string;
	    pinned: boolean;
	    tagIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new BookmarkInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeId = source["themeId"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.favicon = source["favicon"];
	        this.note = source["note"];
	        this.pinned = source["pinned"];
	        this.tagIds = source["tagIds"];
	    }
	}
	export class BookmarkSortInput {
	    ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new BookmarkSortInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ids = source["ids"];
	    }
	}
	export class BrowserBookmark {
	    title: string;
	    url: string;
	    folder: string;
	
	    static createFrom(source: any = {}) {
	        return new BrowserBookmark(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.folder = source["folder"];
	    }
	}
	export class ChangePasswordInput {
	    oldPassword: string;
	    newPassword: string;
	
	    static createFrom(source: any = {}) {
	        return new ChangePasswordInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldPassword = source["oldPassword"];
	        this.newPassword = source["newPassword"];
	    }
	}
	export class CopyTextInput {
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new CopyTextInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class ImportPlanItem {
	    folder: string;
	    targetThemeId: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportPlanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.targetThemeId = source["targetThemeId"];
	    }
	}
	export class ImportPlan {
	    browser: string;
	    items: ImportPlanItem[];
	
	    static createFrom(source: any = {}) {
	        return new ImportPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.items = this.convertValues(source["items"], ImportPlanItem);
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
	
	export class ImportPreview {
	    browser: string;
	    items: BrowserBookmark[];
	    folders: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.items = this.convertValues(source["items"], BrowserBookmark);
	        this.folders = source["folders"];
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
	export class ImportResult {
	    imported: number;
	    skipped: number;
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported = source["imported"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	export class ImportThemeResult {
	    themeName: string;
	    themeId: number;
	    imported: number;
	    skipped: number;
	    tagCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportThemeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.themeName = source["themeName"];
	        this.themeId = source["themeId"];
	        this.imported = source["imported"];
	        this.skipped = source["skipped"];
	        this.tagCount = source["tagCount"];
	    }
	}
	export class OpenURLInput {
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenURLInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	    }
	}
	export class ParsedURL {
	    rawUrl: string;
	    title: string;
	    host: string;
	
	    static createFrom(source: any = {}) {
	        return new ParsedURL(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rawUrl = source["rawUrl"];
	        this.title = source["title"];
	        this.host = source["host"];
	    }
	}
	export class RevealResult {
	    id: number;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new RevealResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.password = source["password"];
	    }
	}
	export class SortInput {
	    ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new SortInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ids = source["ids"];
	    }
	}
	export class TagInput {
	    name: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new TagInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class ThemeInput {
	    name: string;
	    color: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	    }
	}
	export class ThemeSummary {
	    id: number;
	    name: string;
	    color: string;
	    icon: string;
	    sort: number;
	    // Go type: time
	    createdAt: any;
	    bookmarkCount: number;
	    accountCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ThemeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	        this.sort = source["sort"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.bookmarkCount = source["bookmarkCount"];
	        this.accountCount = source["accountCount"];
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
	export class UnlockInput {
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new UnlockInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.password = source["password"];
	    }
	}
	export class VaultStatus {
	    setupRequired: boolean;
	    unlocked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VaultStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.setupRequired = source["setupRequired"];
	        this.unlocked = source["unlocked"];
	    }
	}

}

export namespace bookmarks {
	
	export class Status {
	    browser: string;
	    name: string;
	    running: boolean;
	    installed: boolean;
	    bookmarks: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.name = source["name"];
	        this.running = source["running"];
	        this.installed = source["installed"];
	        this.bookmarks = source["bookmarks"];
	        this.error = source["error"];
	    }
	}

}

export namespace store {
	
	export class Tag {
	    id: number;
	    themeId: number;
	    name: string;
	    color: string;
	    sort: number;
	
	    static createFrom(source: any = {}) {
	        return new Tag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.themeId = source["themeId"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.sort = source["sort"];
	    }
	}
	export class Bookmark {
	    id: number;
	    themeId: number;
	    title: string;
	    url: string;
	    favicon: string;
	    note: string;
	    pinned: boolean;
	    source: string;
	    tags: Tag[];
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Bookmark(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.themeId = source["themeId"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.favicon = source["favicon"];
	        this.note = source["note"];
	        this.pinned = source["pinned"];
	        this.source = source["source"];
	        this.tags = this.convertValues(source["tags"], Tag);
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	
	export class Theme {
	    id: number;
	    name: string;
	    color: string;
	    icon: string;
	    sort: number;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Theme(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	        this.sort = source["sort"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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

