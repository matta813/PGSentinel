export interface RuleProfileEntry { ruleId: string; value: number }
export interface RuleProfile { id: string; name: string; description: string; entries: RuleProfileEntry[]; createdAt: string; updatedAt: string }
export interface RuleProfilesResponse { items: RuleProfile[]; specs: Record<string,{label:string;min:number;max:number;default:number;unit:string}> }
