!/bin/bash                                                                                                                                                                                                           
  # auto-nudge.sh - Nudge all agents every 5 minutes                                                                                                                                                                    
                                                                                                                                                                                                                        
  cd /Users/puneetrai/Projects/bc                                                                                                                                                                                       
                                                                                                                                                                                                                        
  while true; do                                                                                                                                                                                                        
    echo "[$(date '+%H:%M:%S')] Nudging all agents..."                                                                                                                                                                  
    bc channel send all "nudge"                                                                                                                                                                                         
    sleep 300  # 5 minutes                                                                                                                                                                                              
  done
