export namespace main {
	
	export class AccountView {
	    id: string;
	    name: string;
	    profilePath: string;
	    status: string;
	    createdAt: string;
	    lastUsedAt?: string;
	    error?: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccountView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.profilePath = source["profilePath"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.error = source["error"];
	        this.active = source["active"];
	    }
	}

}

