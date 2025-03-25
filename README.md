# worklog-app
A new worklog for my work

This version of the app a gui using bubbletea 
It is a local based versions with a local database (sqlite)

# How to use the worklog
Navigate the worklog with arrow keys for entry field and tab/enter to interact/navigate with pages.

Ctrl+C on any page will close the app so make sure to save all data before closing.

## New View 
### How to enter new view?
Default view when the app starts

### What to do in New View?
This is where new entrys are created and notes are added.
Hit the Save button to add the entry to the database
**Tab** will move between the New and List View when continually pressed 

## List View
### How to enter list view?
**Tab** from New or Modify view to get to the list view.
**Tab** will move between the New and List View when continually pressed

### What to do in list view 
You can view all the current items that have been added to the DB. The list view will start with the latest 10 items and infinite scrol until the last item is reached

## Summary View
### How to enter summary view?
Press ctrl + p to enter summary view when in the list view. 
Select the dates you want to view and then press ctrl + p again.

### What to do in summary view
Here you can see all entries from the last 7 days.
If all these entries seem correct you can hit **Enter** to start the upload process. 
First you will be prompted to login and then link any project codes to scoro tasks that are currently unlinked.
Once complete hit **Enter** again from the summary page to upload all entries accumulated from the week.

## Modify View
### How to enter modify view?
Press enter on an item in the list view. 

### What to do in modify view?
In this view you are able to edit an entry and then save it or delete it if no longer required. 
You are also able to upload a single entry or unlink it if the scoro task bucket has changed and needs to be updated. 
***When a time entry is submitted to an old task, the consequences are untested an unknown. This could range from a http failure response to a success with a new entry being lost to the void. Please check all data is accurate and appropriately organised before submitting. If an issue occurs contact your scoro admin***

## Login 
### Why login? 
The login form takes in the users Scoro username and password and uses this to get a user_token, this is to avoid needing the api_key which some businesses might not want to provide to workers. 

The login data is not saved at all and is only used to obtain auth. The user will need to login on every new session and may need to re-enter details if auth is considered stale in a long session. 

If you do not want to enter login details, a `user.env` file can be used to store these details for the worklog app to read from when required. 

```
SCOROUSER=jeremy.rodarellis@boostdesign.com.au
SCOROPASSWORD=abc123
```

If this doesn't exist then the user will need to use the login form when prompted to upload

## Notes
### What are these used for 
Some managers would prefer a short recap of the action taken in a time entry period for reporting. Some users want to document important notes and task issues/fixes. The notes portions will bridge that gap, the notes are a personal optional note taking field associated with the entry that will be saved but will not be uploaded with the entry when uploading to scoro, this will continue to be the description.

To switch to the notes view, Windows users can press Ctrl + Shift + Right, the left arrow key in the combination will take you back to the new entry view. Mac users can use shift + left/right (this also works on windows). 

## Uploading
User presses upload on an entry. If the event_id is unknown, the user will be prompted to select a Project/Event for all future project code (currently only for the current instance)
This view should first prompt the user to enter their username and passwrod for scoro in to get a user token, another option can be using an env file to get the required details.

If the Scoro task/bucket has changed after a Project code has been linked, the user can unlink and relink to a new task.

## Time resetting
A company may decide to change or update the bucket in which the hours get stored, this will mean that a linked proj code to event_id will be outdated and upload to the wrong place. A check on the first of every month (usually when reporting will occur) will pull a new list of tasks the user is assigned to and ensure that any links still exist in the task list. If they do not they are deleted and any submissions to that project code will need to be relinked when uploading. 

### Currently working on

### Future Additions 
- error log file rotations?
- Filepicker for importing.
- Add inbuilt search functionality based on any input
- Add more testing to code

### Known Bugs

