#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;
in vec2 fragShadowTexCoord;
in float fragShadowDepth;

uniform sampler2D texture0;
uniform sampler2D shadowMap;
uniform vec4 colDiffuse;
uniform vec3 viewPos;
uniform vec4 ambient;
uniform vec3 lightDir;
uniform vec4 lightColor;

out vec4 finalColor;

float calculateShadow()
{
    // Check if fragment is outside shadowmap bounds
    if (fragShadowTexCoord.x < 0.0 || fragShadowTexCoord.x > 1.0 ||
        fragShadowTexCoord.y < 0.0 || fragShadowTexCoord.y > 1.0) {
        return 1.0; // No shadow outside the shadowmap
    }

    // Sample depth from shadowmap
    float shadowDepth = texture(shadowMap, fragShadowTexCoord).r;

    // Bias to prevent shadow acne (adjust as needed)
    float bias = 0.005;

    // Compare depths: if fragment is further than shadowmap sample, it's in shadow
    // Subtract bias from fragShadowDepth to prevent self-shadowing
    if (fragShadowDepth - bias > shadowDepth) {
        return 0.3; // In shadow - return shadow intensity
    }

    return 1.0; // Fully lit
}

void main()
{
    vec3 normal = normalize(fragNormal);
    vec3 viewDir = normalize(viewPos - fragPosition);
    // lightDir is direction light points (e.g. down), we need direction TOWARD light
    vec3 lightDirection = normalize(-lightDir);

    // Calculate shadow factor
    float shadow = calculateShadow();

    // Diffuse lighting (wrap lighting for softer look)
    float NdotL = dot(normal, lightDirection);
    float diff = max(NdotL, 0.0);
    // Add slight wrap lighting for softer shadows on geometry
    float wrapDiff = max(NdotL * 0.5 + 0.5, 0.0);
    diff = mix(diff, wrapDiff, 0.3);

    // Specular (Blinn-Phong) - tighter, brighter highlights
    vec3 halfwayDir = normalize(lightDirection + viewDir);
    float NdotH = max(dot(normal, halfwayDir), 0.0);
    float spec = pow(NdotH, 64.0);

    // Fresnel effect - brighter edges
    float fresnel = pow(1.0 - max(dot(normal, viewDir), 0.0), 3.0);

    // Rim lighting
    float rim = fresnel * max(dot(normal, lightDirection), 0.0);

    // Combine lighting components - apply shadow to diffuse and specular
    vec3 diffuseLight = diff * lightColor.rgb * shadow;
    vec3 specularLight = spec * lightColor.rgb * 1.2 * shadow;
    vec3 rimLight = rim * lightColor.rgb * 0.5 * shadow;
    vec3 fresnelLight = fresnel * ambient.rgb * 0.3;

    // Final lighting (ambient is not affected by shadow)
    vec3 lighting = ambient.rgb + diffuseLight + specularLight + rimLight + fresnelLight;

    // Apply material color
    vec3 baseColor = colDiffuse.rgb * fragColor.rgb;
    vec3 result = lighting * baseColor;

    // Slight tone mapping to prevent over-bright
    result = result / (result + vec3(1.0));

    // Gamma correction
    result = pow(result, vec3(1.0/2.2));

    finalColor = vec4(result, colDiffuse.a);
}
