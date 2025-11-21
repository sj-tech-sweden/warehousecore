## Project Ideas

### RentalManagement Tool
- Goal to build a highly professional and visually appealing rental management application
- Focus on consistent design theme across the entire app (font, color scheme, overall perception)
- Prioritize fast website loading performance

## Configuration Examples

### Database Configuration Template
- External database hosted at your-database-host.example.com
- Username: tsweb
- Password: <configured_in_env>
- Database: RentalCore
- Host:     app.example.com
## Project Resources

### Database Templates
- Das aktuelle Datenbank template liegt im root der Codebase und heißt RentalCore.sql

## Professional Development Mindset

### Software Development Philosophy
- Du bist ein höchstprofessioneller Softwaredeveloper, welcher niemals sagt, dass ein Programm fertig wär oder funktionieren würde, obwohl es das nicht zu 100% tut.

### File Management
- Wenn du temporäre Dateien oder Debug Dateien anlegst, dann löschst du die Dateien sofort, nachdem du sie nicht mehr brauchst. Auch wenn du sowas wie eine neue Version einer vorhandenen Datei anlegst, dann wird die alte sofort gelöscht.

## Development Workflow

### Server Management
- Please restart the Server always, after you made changes, so i dont have to.
- NEVER use the command: pkill server because it closes my tmux session which is NOT WANTED
- ALWAYS build and push the project to docker hub image: nbt4/rentalcore and please make the version tag simple like 1.X and so on.
- and always push your changes to github for possible easy rollbacks. and i mean every time you finished smth. push it to github
- please update after every change the README file.
- REMEMBER: NO SENSITIVE DATA IS ALLOWED TO BE PUSHED TO GITHUB!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! CHECK ALL FILES WHICH ARE GONNA BE PUSHED AND REMOVE THERE SENSITIVE DATA AND REPLACE IT WITH DEMO DATA!!!!!!!!!!
- DO NOT DO NEVER WRITE IN COMMIT MESSAGES, THAT THE COMMIT WAS MADE BY CLAUDE OR SMTH. IT SHOULD LOOK LIKE THAT IT ISNT BY AI. ALSO ONLY USE THE NORMAL GIT COMMANDS, SO I DONT HAVE TO CONNECT YOU TO MY REPO
- please keep the navigation menu in the readme uptodate
- my mysql user: warehouse_user, my mysql pw: <configured_in_env>, my mysql host: db.example.com, my Database: rentalcore
- Always push when you are finished editing or so your changes to dockerhub and github. For dockerhub check which version was the latest version pushed to dockerhub and push the following version. Also ALWAYS PUSH THE :latest version
- You may use the existing Docker login on this host for pushes (username: nobentie) and do not re-enter credentials unless needed.
- Always push to GitLab after completing work and keep all README files updated to reflect the latest changes and versions.
- dont let old versions of files there. and dont name updated or fixed versions like [filename]_updated_fixed_complete_CORRECT use normal file names and remove the old files.
- you can make edits on the Database Tables !!!NOT THE DATA!!! on your own
