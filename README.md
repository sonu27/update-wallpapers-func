# Update Wallpapers Function

This function requests wallpapers for different countries using the Bing Wallpapers API.

Wallpapers get added if both the 1920x1080 and the 1920x1200 images exist and are not already in the database (Firestore).

For non-English countries, the descriptions are translated into English using the Google Translate API.

Finally, the images are annotated using the Google Vision API. This allows wallpapers to be searched by these labels in the future and provide better SEO.

If previously stored, translated English wallpapers appear again but in English, then the descriptions are replaced but original date is kept.

There is also splitting of the description to get the title and copyright.

## Improvements
- Separate the functions into different packages and enable CI/CD. They were originally kept in one file for ease of deployment.
- TEST!
- Merge with API source code (maybe).
- Use concurrency for making requests (maybe).
- Download wallpaper even if large image is not found.
- Use IAM roles instead of service account keys.
