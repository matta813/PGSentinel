import {Activity,AlertTriangle,Database,FileSearch,Home,KeyRound,Lock,LogOut,Menu,Search,Settings,Table2,PanelLeftClose,Sun,Moon} from 'lucide-react'
import {NavLink,Outlet} from 'react-router-dom'
import {useEffect,useState} from 'react'
import {api} from '../api/client'
import {useApi} from '../hooks/useApi'

const items=[['/','Overview',Home],['/problems','Problems',AlertTriangle],['/servers','Servers',Database],['/queries','Queries',Search],['/tables','Tables',Table2],['/indexes','Indexes',KeyRound],['/vacuum','Vacuum',Activity],['/locks','Locks',Lock],['/settings','Settings',Settings]] as const

export function AppLayout({onLogout}:{onLogout:()=>void}){
  const[open,setOpen]=useState(false)
  const[dark,setDark]=useState(()=>localStorage.theme!=='light')
  const{data:build}=useApi(()=>api.get<{version:string;commit:string}>('/version'),[])
  useEffect(()=>{document.documentElement.dataset.theme=dark?'dark':'light';localStorage.theme=dark?'dark':'light'},[dark])
  async function logout(){await api.post('/auth/logout');onLogout()}
  return <div className="shell"><aside className={open?'open':''}><div className="brand"><div className="logo"><FileSearch size={19}/></div><strong>pgsentinel</strong><button className="icon mobile" onClick={()=>setOpen(false)}><PanelLeftClose/></button></div><nav>{items.map(([to,label,Icon])=><NavLink key={to} to={to} end={to==='/'} onClick={()=>setOpen(false)}><Icon size={17}/>{label}</NavLink>)}</nav><div className="side-foot"><div><span className="live-dot"/>Monitoring active</div><small>PGSentinel {build?.version??'dev'}{build?.commit!=='unknown'&&build?.commit?` · ${build.commit.slice(0,8)}`:''}</small></div></aside><main><header><button className="icon menu" onClick={()=>setOpen(true)}><Menu/></button><div><span className="crumb">PostgreSQL health analysis</span></div><div><button className="icon" title="Toggle color theme" onClick={()=>setDark(v=>!v)}>{dark?<Sun/>:<Moon/>}</button><button className="icon" title="Sign out" onClick={logout}><LogOut/></button></div></header><div className="page"><Outlet/></div></main></div>
}
