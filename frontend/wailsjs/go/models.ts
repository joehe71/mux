export namespace main {
	
	export class AccountView {
	    id: string;
	    name: string;
	    email?: string;
	    avatarUrl?: string;
	    planType?: string;
	    profilePath: string;
	    status: string;
	    createdAt: string;
	    lastUsedAt?: string;
	    error?: string;
	    // Go type: accounts
	    usage?: any;
	    usageUpdatedAt?: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccountView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.email = source["email"];
	        this.avatarUrl = source["avatarUrl"];
	        this.planType = source["planType"];
	        this.profilePath = source["profilePath"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.error = source["error"];
	        this.usage = this.convertValues(source["usage"], null);
	        this.usageUpdatedAt = source["usageUpdatedAt"];
	        this.active = source["active"];
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

